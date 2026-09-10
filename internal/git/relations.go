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
}

// LineRelations is archaeology context for one blamed line.
type LineRelations struct {
	Line     BlameLine
	Path     string
	Rev      string
	History  []Commit // recent commits that touched the path (follow)
	Previous *Commit  // previous blame commit when known
}

// Relations returns parents, children, and files for hash.
func (r *Repo) Relations(ctx context.Context, hash string) (CommitRelations, error) {
	var out CommitRelations
	detail, err := r.Show(ctx, hash)
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
	return out, nil
}

// LineRelationsAt builds neighbourhood for a blame line at path@rev.
func (r *Repo) LineRelationsAt(ctx context.Context, path, rev string, line BlameLine) (LineRelations, error) {
	out := LineRelations{Line: line, Path: path, Rev: rev}
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
