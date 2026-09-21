package git

import (
	"context"
	"fmt"
	"strings"
)

// CommitRelations is the graph neighbourhood of one commit.
type CommitRelations struct {
	Hash     string
	Subject  string
	Author   string
	Email    string
	Parents  []Commit
	Children []Commit
	Files    []FileChange
	HotSpots []CoChange // paths that often change with this commit's files
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

// childHashes returns direct children of hash in this repo.
func (r *Repo) childHashes(ctx context.Context, hash string) ([]string, error) {
	fullOut, err := r.run(ctx, "rev-parse", hash)
	if err != nil {
		return nil, err
	}
	full := strings.TrimSpace(string(fullOut))

	// rev-list HASH only walks ancestors; --all --children is needed for kids.
	out, err := r.run(ctx, "rev-list", "--children", "--all")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == full || strings.HasPrefix(full, fields[0]) || strings.HasPrefix(fields[0], full) {
			if len(fields) == 1 {
				return nil, nil
			}
			return fields[1:], nil
		}
	}
	return nil, nil
}

// ChildrenOf returns direct child commit hashes of hash (may be empty).
func (r *Repo) ChildrenOf(ctx context.Context, hash string) ([]string, error) {
	return r.childHashes(ctx, hash)
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
