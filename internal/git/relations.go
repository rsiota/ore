package git

import (
	"context"
	"fmt"
	"strings"
)

// CommitRelations is the graph neighbourhood of one commit.
type CommitRelations struct {
	Hash      string
	Subject   string
	Author    string
	Email     string
	Parents   []Commit
	Children  []Commit
	Files     []FileChange
	HotSpots  []CoChange    // paths that often change with this commit's files
	Ownership []AuthorShare // quiet author mix for the commit's paths
	OwnSample int           // commits examined for Ownership (sample, not top-N)
}

// LineRelations is archaeology context for one blamed line.
type LineRelations struct {
	Line         BlameLine
	Path         string
	Rev          string
	History      []PathCommit // recent commits that touched the path (follow + edges)
	Previous     *Commit      // previous blame commit when known
	PreviousPath string       // porcelain previous path when it differs / is set
	HotSpots     []CoChange
	Ownership    []AuthorShare // quiet author mix from path history
	OwnSample    int           // authors/commits examined for Ownership
}

// Relations returns parents, children, and files for hash.
func (r *Repo) Relations(ctx context.Context, hash string) (CommitRelations, error) {
	var out CommitRelations
	detail, err := r.ShowHeader(ctx, hash, "")
	if err != nil {
		return out, err
	}
	out.Hash = detail.Commit.Hash
	out.Subject = detail.Commit.Subject
	out.Author = detail.Commit.Author
	out.Email = detail.Commit.Email
	out.Files = detail.Files

	for _, p := range detail.Commit.Parents {
		c, err := r.commitSummary(ctx, p)
		if err != nil {
			continue
		}
		out.Parents = append(out.Parents, c)
	}

	childHashes, err := r.childHashes(ctx, hash)
	if err != nil {
		return out, err
	}
	for _, ch := range childHashes {
		c, err := r.commitSummary(ctx, ch)
		if err != nil {
			continue
		}
		out.Children = append(out.Children, c)
	}

	seeds := make([]string, 0, len(detail.Files)*2)
	for _, f := range detail.Files {
		seeds = append(seeds, f.Path)
		if f.OldPath != "" {
			seeds = append(seeds, f.OldPath)
		}
	}
	if hot, err := r.CoChangedFiles(ctx, seeds, out.Hash); err == nil {
		out.HotSpots = hot
	}
	if own, sample, err := r.PathOwnership(ctx, out.Hash, seeds, OwnershipSample, OwnershipTop); err == nil {
		out.Ownership = own
		out.OwnSample = sample
	}
	return out, nil
}

// LineRelationsAt builds neighbourhood for a blame line at path@rev.
func (r *Repo) LineRelationsAt(ctx context.Context, path, rev string, line BlameLine) (LineRelations, error) {
	out := LineRelations{Line: line, Path: path, Rev: rev, PreviousPath: line.PreviousPath}
	hist, err := r.FileHistory(ctx, path, LogOptions{MaxCount: 30, Rev: rev})
	if err != nil {
		return out, err
	}
	out.History = hist
	if line.PreviousHash != "" {
		c, err := r.commitSummary(ctx, line.PreviousHash)
		if err == nil {
			out.Previous = &c
		}
	}
	seeds := []string{path}
	if line.PreviousPath != "" && line.PreviousPath != path {
		seeds = append(seeds, line.PreviousPath)
	}
	if hot, err := r.CoChangedFiles(ctx, seeds, ""); err == nil {
		out.HotSpots = hot
	}
	names := make([]string, 0, len(hist)+1)
	if line.Author != "" {
		names = append(names, line.Author)
	}
	for _, c := range hist {
		if c.Author != "" {
			names = append(names, c.Author)
		}
	}
	out.Ownership = SummarizeAuthors(names, OwnershipTop)
	out.OwnSample = 0
	for _, n := range names {
		if strings.TrimSpace(n) != "" {
			out.OwnSample++
		}
	}
	return out, nil
}

func (r *Repo) commitSummary(ctx context.Context, hash string) (Commit, error) {
	format := strings.Join([]string{"%H", "%h", "%an", "%ae", "%aI", "%s", "%P"}, fieldSep)
	out, err := r.run(ctx, "show", "-s", "--format="+format, hash)
	if err != nil {
		return Commit{}, err
	}
	commits, err := parseCommitLog([]byte(strings.TrimSpace(string(out)) + recSep))
	if err != nil {
		return Commit{}, err
	}
	if len(commits) == 0 {
		return Commit{}, fmt.Errorf("no commit %s", hash)
	}
	return commits[0], nil
}

// ChildrenAmong returns hashes in commits that list hash as a parent
// (newest-first when commits is newest-first). Used to hop `c` without git.
func ChildrenAmong(commits []Commit, hash string) []string {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil
	}
	var kids []string
	for _, c := range commits {
		for _, p := range c.Parents {
			if hashMatch(p, hash) {
				kids = append(kids, c.Hash)
				break
			}
		}
	}
	return kids
}

// ResetGraph drops the cached children map (call after the repo may have moved).
func (r *Repo) ResetGraph() {
	if r == nil {
		return
	}
	r.graphMu.Lock()
	r.childByHash = nil
	r.graphMu.Unlock()
}

// childHashes returns direct children of hash in this repo (cached after first walk).
func (r *Repo) childHashes(ctx context.Context, hash string) ([]string, error) {
	full, err := r.revParseFull(ctx, hash)
	if err != nil {
		return nil, err
	}
	graph, err := r.childrenGraph(ctx)
	if err != nil {
		return nil, err
	}
	if kids, ok := graph[full]; ok {
		return append([]string(nil), kids...), nil
	}
	for k, v := range graph {
		if hashMatch(k, full) {
			return append([]string(nil), v...), nil
		}
	}
	return nil, nil
}

func (r *Repo) revParseFull(ctx context.Context, hash string) (string, error) {
	out, err := r.run(ctx, "rev-parse", hash)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repo) childrenGraph(ctx context.Context) (map[string][]string, error) {
	r.graphMu.Lock()
	defer r.graphMu.Unlock()
	if r.childByHash != nil {
		return r.childByHash, nil
	}
	// One --all walk per session; ResetGraph after refresh.
	out, err := r.run(ctx, "rev-list", "--children", "--all")
	if err != nil {
		return nil, err
	}
	m := make(map[string][]string)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) == 1 {
			m[fields[0]] = nil
			continue
		}
		m[fields[0]] = append([]string(nil), fields[1:]...)
	}
	r.childByHash = m
	return m, nil
}

// ChildrenOf returns direct child commit hashes of hash (may be empty).
func (r *Repo) ChildrenOf(ctx context.Context, hash string) ([]string, error) {
	return r.childHashes(ctx, hash)
}

// ChildrenToward returns direct children of hash that lie on the path to tip
// (typically HEAD / the viewed branch). Cheaper than ChildrenOf on a large repo.
func (r *Repo) ChildrenToward(ctx context.Context, hash, tip string) ([]string, error) {
	if tip == "" {
		tip = "HEAD"
	}
	full, err := r.revParseFull(ctx, hash)
	if err != nil {
		return nil, err
	}
	out, err := r.run(ctx, "rev-list", "--parents", full+".."+tip)
	if err != nil {
		return nil, err
	}
	var kids []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		commit := fields[0]
		for _, p := range fields[1:] {
			if hashMatch(p, full) {
				kids = append(kids, commit)
				break
			}
		}
	}
	return kids, nil
}

// MergeBase returns the best common ancestor of a and b.
func (r *Repo) MergeBase(ctx context.Context, a, b string) (string, error) {
	if a == "" || b == "" {
		return "", fmt.Errorf("merge-base needs two revisions")
	}
	out, err := r.run(ctx, "merge-base", a, b)
	if err != nil {
		return "", err
	}
	h := strings.TrimSpace(string(out))
	if h == "" {
		return "", fmt.Errorf("no merge-base for %s and %s", short(a), short(b))
	}
	return h, nil
}

func short(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
