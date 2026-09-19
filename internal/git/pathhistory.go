package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// PathCommit is one hop in a followed path history: the commit plus the path
// name at that revision and optional rename/copy edge metadata.
type PathCommit struct {
	Commit
	Path    string // path as of this commit
	OldPath string // source path when Status is rename/copy
	Status  string // A, M, D, R, C, … (git name-status; R/C may include score)
}

// EdgeLabel returns a short archaeology label for rename/copy hops.
// Empty when this commit is not a rename/copy of the followed path.
func (p PathCommit) EdgeLabel() string {
	if p.OldPath == "" {
		return ""
	}
	st := p.Status
	switch {
	case st == "R" || strings.HasPrefix(st, "R"):
		return "moved from " + p.OldPath
	case st == "C" || strings.HasPrefix(st, "C"):
		return "copied from " + p.OldPath
	default:
		if p.OldPath != "" {
			return "moved from " + p.OldPath
		}
	}
	return ""
}

// PathLabel is the path column text: tip name, or "new (moved from old)" on hops.
func (p PathCommit) PathLabel() string {
	if edge := p.EdgeLabel(); edge != "" && p.Path != "" {
		return p.Path + " (" + edge + ")"
	}
	if p.Path != "" {
		return p.Path
	}
	return p.OldPath
}

// FileHistory returns commits that touched path, following renames (--follow),
// with per-commit path and rename/copy edges from name-status.
func (r *Repo) FileHistory(ctx context.Context, path string, opt LogOptions) ([]PathCommit, error) {
	if path == "" {
		return nil, fmt.Errorf("path required")
	}
	opt.Path = path
	opt.Follow = true
	return r.pathCommitLog(ctx, opt)
}

func (r *Repo) pathCommitLog(ctx context.Context, opt LogOptions) ([]PathCommit, error) {
	limit := opt.MaxCount
	if limit <= 0 {
		limit = defaultLogLimit
	}
	rev := opt.Rev
	if rev == "" {
		rev = "HEAD"
	}
	if opt.Follow && opt.Path == "" {
		return nil, fmt.Errorf("Follow requires Path")
	}

	format := strings.Join([]string{
		"%H", "%h", "%an", "%ae", "%aI", "%s", "%P",
	}, fieldSep) + recSep

	args := []string{"log", rev}
	if opt.Follow {
		args = append(args, "--follow")
	}
	args = append(args,
		"--name-status",
		"--find-renames",
		"--date=iso-strict",
		"--format="+format,
		"-n", strconv.Itoa(limit),
	)
	if opt.Path != "" {
		args = append(args, "--", opt.Path)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parsePathCommitLog(out, opt.Path)
}

func parsePathCommitLog(out []byte, tipPath string) ([]PathCommit, error) {
	raw := string(out)
	if raw == "" {
		return nil, nil
	}
	// git log --name-status --format=…%x1e emits:
	//   <meta>\x1e
	//   <name-status…>
	//   <meta>\x1e
	//   <name-status…>
	// so name-status trails each record separator. Scan line-by-line.
	var (
		commits []PathCommit
		cur     *PathCommit
	)
	flush := func() {
		if cur == nil {
			return
		}
		if cur.Path == "" {
			cur.Path = tipPath
		}
		commits = append(commits, *cur)
		cur = nil
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Meta records end with recSep; strip it when present.
		if strings.Contains(line, string(recSep)) {
			line = strings.TrimSuffix(line, string(recSep))
			line = strings.TrimSpace(line)
		}
		if line == "" {
			continue
		}
		if strings.Contains(line, fieldSep) {
			flush()
			c, err := parseCommitMeta(line)
			if err != nil {
				return nil, err
			}
			pc := PathCommit{Commit: c, Path: tipPath}
			cur = &pc
			continue
		}
		st, path, old, ok := parseNameStatusLine(line)
		if !ok {
			continue
		}
		if cur == nil {
			continue
		}
		if cur.Status == "" {
			cur.Status = st
			cur.Path = path
			cur.OldPath = old
		}
	}
	flush()
	return commits, nil
}

func parseCommitMeta(rec string) (Commit, error) {
	parts := strings.Split(rec, fieldSep)
	if len(parts) < 7 {
		return Commit{}, fmt.Errorf("unexpected log record: %q", rec)
	}
	ts, err := time.Parse(time.RFC3339, parts[4])
	if err != nil {
		ts, err = time.Parse("2006-01-02T15:04:05-07:00", parts[4])
		if err != nil {
			return Commit{}, fmt.Errorf("parse date %q: %w", parts[4], err)
		}
	}
	var parents []string
	if parts[6] != "" {
		parents = strings.Fields(parts[6])
	}
	return Commit{
		Hash:      parts[0],
		ShortHash: parts[1],
		Author:    parts[2],
		Email:     parts[3],
		Date:      ts,
		Subject:   parts[5],
		Parents:   parents,
	}, nil
}

// parseNameStatusLine parses one git --name-status line.
// Forms: "M\tpath", "A\tpath", "R050\told\tnew", "C100\told\tnew".
func parseNameStatusLine(line string) (status, path, oldPath string, ok bool) {
	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return "", "", "", false
	}
	status = fields[0]
	switch {
	case len(fields) >= 3 && (status == "R" || strings.HasPrefix(status, "R") ||
		status == "C" || strings.HasPrefix(status, "C")):
		return status, fields[2], fields[1], true
	default:
		return status, fields[1], "", true
	}
}

// PathCommitsAsCommits strips path metadata for APIs that take []Commit.
func PathCommitsAsCommits(pcs []PathCommit) []Commit {
	out := make([]Commit, len(pcs))
	for i := range pcs {
		out[i] = pcs[i].Commit
	}
	return out
}
