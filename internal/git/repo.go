// Package git talks to a repository through the git CLI.
//
// Using the system git keeps blame, rename following, and log graph behaviour
// correct without reimplementing them. Swap in a library later only if profiling
// shows the CLI is the bottleneck.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrNotRepository is returned when path is not inside a git work tree.
var ErrNotRepository = errors.New("not a git repository")

// Repo is a handle on a local git work tree.
type Repo struct {
	Path string // absolute path to the work tree root
	Git  string // git binary; empty means "git" on PATH
}

// Open finds the work tree containing path (or path itself) and returns a Repo.
func Open(path string) (*Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		abs = filepath.Dir(abs)
	}

	r := &Repo{Path: abs}
	out, err := r.run(context.Background(), "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotRepository, abs)
	}
	r.Path = strings.TrimSpace(string(out))
	return r, nil
}

func (r *Repo) bin() string {
	if r.Git != "" {
		return r.Git
	}
	return "git"
}

func (r *Repo) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.bin(), args...)
	cmd.Dir = r.Path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

// Commit is one row in the commit grid.
type Commit struct {
	Hash      string
	ShortHash string
	Author    string
	Email     string
	Date      time.Time
	Subject   string
	Parents   []string
	Files     int
	Additions int
	Deletions int
}

// LogOptions controls CommitLog.
type LogOptions struct {
	MaxCount int    // 0 = default cap
	Path     string // optional path limiter
	Rev      string // revision range; empty = HEAD
	Follow   bool   // git log --follow (requires Path)
}

const defaultLogLimit = 500

// field sep + record sep chosen to avoid colliding with commit message text.
const (
	fieldSep = "\x1f"
	recSep   = "\x1e"
)

// CommitLog returns commits newest-first.
func (r *Repo) CommitLog(ctx context.Context, opt LogOptions) ([]Commit, error) {
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
	return parseCommitLog(out)
}

// FileHistory returns commits that touched path, following renames (--follow).
func (r *Repo) FileHistory(ctx context.Context, path string, opt LogOptions) ([]Commit, error) {
	if path == "" {
		return nil, fmt.Errorf("path required")
	}
	opt.Path = path
	opt.Follow = true
	return r.CommitLog(ctx, opt)
}

func parseCommitLog(out []byte) ([]Commit, error) {
	raw := string(out)
	if raw == "" {
		return nil, nil
	}
	records := strings.Split(raw, recSep)
	commits := make([]Commit, 0, len(records))
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.Split(rec, fieldSep)
		if len(parts) < 7 {
			return nil, fmt.Errorf("unexpected log record: %q", rec)
		}
		ts, err := time.Parse(time.RFC3339, parts[4])
		if err != nil {
			// Fall back for odd git date formats.
			ts, err = time.Parse("2006-01-02T15:04:05-07:00", parts[4])
			if err != nil {
				return nil, fmt.Errorf("parse date %q: %w", parts[4], err)
			}
		}
		var parents []string
		if parts[6] != "" {
			parents = strings.Fields(parts[6])
		}
		commits = append(commits, Commit{
			Hash:      parts[0],
			ShortHash: parts[1],
			Author:    parts[2],
			Email:     parts[3],
			Date:      ts,
			Subject:   parts[5],
			Parents:   parents,
		})
	}
	return commits, nil
}

// CommitDetail is a commit plus its message body and unified diff.
type CommitDetail struct {
	Commit  Commit
	Body    string // message without subject line
	Diff    string
	Stat    string
	Files   []FileChange
}

// FileChange is one path touched by a commit.
type FileChange struct {
	Path      string
	OldPath   string // set on renames
	Status    string // A, M, D, R, C, …
	Additions int
	Deletions int
}

// Show returns message + numstat + patch for hash (whole commit).
func (r *Repo) Show(ctx context.Context, hash string) (CommitDetail, error) {
	return r.show(ctx, hash, "")
}

// ShowPath is Show limited to a single path (path-scoped stat + patch).
func (r *Repo) ShowPath(ctx context.Context, hash, path string) (CommitDetail, error) {
	return r.show(ctx, hash, path)
}

func (r *Repo) show(ctx context.Context, hash, path string) (CommitDetail, error) {
	var detail CommitDetail

	metaOut, err := r.run(ctx, "show", "-s",
		"--format="+strings.Join([]string{"%H", "%h", "%an", "%ae", "%aI", "%s", "%P", "%b"}, fieldSep),
		hash)
	if err != nil {
		return detail, err
	}
	line := strings.TrimSpace(string(metaOut))
	parts := strings.SplitN(line, fieldSep, 8)
	if len(parts) < 7 {
		return detail, fmt.Errorf("unexpected show format: %q", line)
	}
	ts, err := time.Parse(time.RFC3339, parts[4])
	if err != nil {
		ts, _ = time.Parse("2006-01-02T15:04:05-07:00", parts[4])
	}
	var parents []string
	if parts[6] != "" {
		parents = strings.Fields(parts[6])
	}
	body := ""
	if len(parts) >= 8 {
		body = strings.TrimRight(parts[7], "\n")
	}
	detail.Commit = Commit{
		Hash:      parts[0],
		ShortHash: parts[1],
		Author:    parts[2],
		Email:     parts[3],
		Date:      ts,
		Subject:   parts[5],
		Parents:   parents,
	}
	detail.Body = body

	numstatArgs := []string{"show", "--format=", "--numstat", "--find-renames", hash}
	diffArgs := []string{"show", "--format=", "--patch", "--find-renames", hash}
	statArgs := []string{"show", "--format=", "--stat", "--find-renames", hash}
	if path != "" {
		numstatArgs = append(numstatArgs, "--", path)
		diffArgs = append(diffArgs, "--", path)
		statArgs = append(statArgs, "--", path)
	}

	statOut, err := r.run(ctx, numstatArgs...)
	if err != nil {
		return detail, err
	}
	files, add, del := parseNumstat(statOut)
	detail.Files = files
	detail.Commit.Files = len(files)
	detail.Commit.Additions = add
	detail.Commit.Deletions = del

	diffOut, err := r.run(ctx, diffArgs...)
	if err != nil {
		return detail, err
	}
	detail.Diff = string(diffOut)

	summaryOut, err := r.run(ctx, statArgs...)
	if err != nil {
		return detail, err
	}
	detail.Stat = string(summaryOut)

	return detail, nil
}

func parseNumstat(out []byte) (files []FileChange, additions, deletions int) {
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		fc := FileChange{Status: "M"}
		if fields[0] == "-" {
			fc.Additions = 0
		} else if n, err := strconv.Atoi(fields[0]); err == nil {
			fc.Additions = n
			additions += n
		}
		if fields[1] == "-" {
			fc.Deletions = 0
		} else if n, err := strconv.Atoi(fields[1]); err == nil {
			fc.Deletions = n
			deletions += n
		}
		pathField := fields[2]
		if len(fields) >= 4 {
			// rename: old \t new (git numstat with -M)
			fc.OldPath = fields[2]
			fc.Path = fields[3]
			fc.Status = "R"
		} else if strings.Contains(pathField, " => ") {
			parts := strings.SplitN(pathField, " => ", 2)
			fc.OldPath = parts[0]
			fc.Path = parts[1]
			fc.Status = "R"
		} else {
			fc.Path = pathField
		}
		files = append(files, fc)
	}
	return files, additions, deletions
}

// HeadShort returns the short HEAD hash, or empty if unavailable.
func (r *Repo) HeadShort(ctx context.Context) string {
	out, err := r.run(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// BranchName returns the current branch, or "DETACHED" / empty.
func (r *Repo) BranchName(ctx context.Context) string {
	out, err := r.run(ctx, "branch", "--show-current")
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "DETACHED"
	}
	return name
}
