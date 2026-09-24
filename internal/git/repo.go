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
	"sync"
	"time"
)

// ErrNotRepository is returned when path is not inside a git work tree.
var ErrNotRepository = errors.New("not a git repository")

// Repo is a handle on a local git work tree.
type Repo struct {
	Path string // absolute path to the work tree root
	Git  string // git binary; empty means "git" on PATH

	graphMu     sync.Mutex
	childByHash map[string][]string // full hash → direct children (session cache)
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

// DefaultLogLimit is the silent window for CommitLog / FileHistory when MaxCount is 0.
const DefaultLogLimit = 500

// field sep + record sep chosen to avoid colliding with commit message text.
const (
	fieldSep = "\x1f"
	recSep   = "\x1e"
)

// CommitLog returns commits newest-first.
func (r *Repo) CommitLog(ctx context.Context, opt LogOptions) ([]Commit, error) {
	limit := opt.MaxCount
	if limit <= 0 {
		limit = DefaultLogLimit
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
		c, err := parseCommitMeta(rec)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}
	return commits, nil
}

// CommitDetail is a commit plus its message body and unified diff.
type CommitDetail struct {
	Commit Commit
	Body   string // message without subject line
	Diff   string
	Stat   string
	Files  []FileChange
}

// FileChange is one path touched by a commit.
type FileChange struct {
	Path      string
	OldPath   string // set on renames
	Status    string // A, M, D, R, C, …
	Additions int
	Deletions int
}

// showMetaFormat is machine-readable metadata; body is terminated by recSep so
// a following --numstat/--stat block can be parsed from the same git show.
const showMetaFormat = "%H%x1f%h%x1f%an%x1f%ae%x1f%aI%x1f%s%x1f%P%x1f%b%x1e"

// Show returns message + numstat + patch for hash (whole commit).
// unified is the git -U context size (clamped; use 3 for the usual default).
func (r *Repo) Show(ctx context.Context, hash string, unified int) (CommitDetail, error) {
	return r.show(ctx, hash, "", unified)
}

// ShowPath is Show limited to a single path (path-scoped stat + patch).
func (r *Repo) ShowPath(ctx context.Context, hash, path string, unified int) (CommitDetail, error) {
	return r.show(ctx, hash, path, unified)
}

// ShowHeader returns metadata, numstat, and --stat in one git show (no patch).
func (r *Repo) ShowHeader(ctx context.Context, hash, path string) (CommitDetail, error) {
	var detail CommitDetail
	args := []string{"show", "--format=" + showMetaFormat, "--numstat", "--stat", "--find-renames", hash}
	if path != "" {
		args = append(args, "--", path)
	}
	out, err := r.run(ctx, args...)
	if err != nil {
		return detail, err
	}
	detail, rest, err := parseShowHeader(out)
	if err != nil {
		return detail, err
	}
	files, add, del, stat := parseNumstatAndStat(rest)
	detail.Files = files
	detail.Commit.Files = len(files)
	detail.Commit.Additions = add
	detail.Commit.Deletions = del
	detail.Stat = stat
	return detail, nil
}

// ShowPatch returns the unified patch only (one git show --patch).
func (r *Repo) ShowPatch(ctx context.Context, hash, path string, unified int) (string, error) {
	if unified < 0 {
		unified = 3
	}
	args := []string{"show", "--format=", "--patch", "--find-renames", fmt.Sprintf("-U%d", unified), hash}
	if path != "" {
		args = append(args, "--", path)
	}
	out, err := r.run(ctx, args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (r *Repo) show(ctx context.Context, hash, path string, unified int) (CommitDetail, error) {
	detail, err := r.ShowHeader(ctx, hash, path)
	if err != nil {
		return detail, err
	}
	diff, err := r.ShowPatch(ctx, hash, path, unified)
	if err != nil {
		return detail, err
	}
	detail.Diff = diff
	return detail, nil
}

func parseShowHeader(out []byte) (CommitDetail, []byte, error) {
	var detail CommitDetail
	end := bytes.Index(out, []byte(recSep))
	if end < 0 {
		return detail, nil, fmt.Errorf("unexpected show format: missing record separator")
	}
	parts := strings.SplitN(string(out[:end]), fieldSep, 8)
	if len(parts) < 7 {
		return detail, nil, fmt.Errorf("unexpected show format: %q", string(out[:end]))
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
	return detail, out[end+len(recSep):], nil
}

func parseNumstatAndStat(rest []byte) (files []FileChange, additions, deletions int, stat string) {
	text := strings.TrimLeft(string(rest), "\r\n")
	if text == "" {
		return nil, 0, 0, ""
	}
	var numstat strings.Builder
	var statBuf strings.Builder
	inStat := false
	for _, line := range strings.Split(text, "\n") {
		if !inStat {
			if line == "" {
				continue
			}
			if isNumstatLine(line) {
				numstat.WriteString(line)
				numstat.WriteByte('\n')
				continue
			}
			inStat = true
		}
		statBuf.WriteString(line)
		statBuf.WriteByte('\n')
	}
	files, additions, deletions = parseNumstat([]byte(numstat.String()))
	return files, additions, deletions, statBuf.String()
}

func isNumstatLine(line string) bool {
	fields := strings.Split(line, "\t")
	if len(fields) < 3 {
		return false
	}
	return isNumstatCount(fields[0]) && isNumstatCount(fields[1])
}

func isNumstatCount(s string) bool {
	if s == "-" {
		return true
	}
	_, err := strconv.Atoi(s)
	return err == nil
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

// RevCount returns the number of commits reachable from rev (optionally
// limited to path, without --follow). rev empty means HEAD.
func (r *Repo) RevCount(ctx context.Context, rev, path string) (int, error) {
	if rev == "" {
		rev = "HEAD"
	}
	args := []string{"rev-list", "--count", rev}
	if path != "" {
		args = append(args, "--", path)
	}
	out, err := r.run(ctx, args...)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("rev-list --count: %w", err)
	}
	return n, nil
}

// HeadShort returns the short HEAD hash, or empty if unavailable.
func (r *Repo) HeadShort(ctx context.Context) string {
	return r.RevShort(ctx, "HEAD")
}

// RevShort returns the short hash for rev, or empty if unavailable.
func (r *Repo) RevShort(ctx context.Context, rev string) string {
	if rev == "" {
		rev = "HEAD"
	}
	out, err := r.run(ctx, "rev-parse", "--short", rev)
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

// Ref is a local or remote branch tip (read-only view target).
type Ref struct {
	Name    string // short name: main, origin/feature
	Hash    string // short object name
	Subject string
	Current bool // worktree HEAD points here
	Remote  bool // refs/remotes/…
}

// ListBranches returns local then remote branches (newest tip first within each).
func (r *Repo) ListBranches(ctx context.Context) ([]Ref, error) {
	format := strings.Join([]string{
		"%(refname)",
		"%(refname:short)",
		"%(objectname:short)",
		"%(subject)",
		"%(HEAD)",
	}, fieldSep)
	out, err := r.run(ctx, "for-each-ref",
		"--sort=-committerdate",
		"--format="+format,
		"refs/heads",
		"refs/remotes",
	)
	if err != nil {
		return nil, err
	}
	var (
		local  []Ref
		remote []Ref
	)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, fieldSep, 5)
		if len(parts) < 4 {
			continue
		}
		full, short, hash, subject := parts[0], parts[1], parts[2], parts[3]
		headMark := ""
		if len(parts) >= 5 {
			headMark = parts[4]
		}
		ref := Ref{
			Name:    short,
			Hash:    hash,
			Subject: subject,
			Current: headMark == "*",
			Remote:  strings.HasPrefix(full, "refs/remotes/"),
		}
		// Skip remote HEAD symbolic aliases (origin/HEAD).
		if ref.Remote && (strings.HasSuffix(full, "/HEAD") || short == "origin/HEAD" || strings.HasSuffix(short, "/HEAD")) {
			continue
		}
		if ref.Remote {
			remote = append(remote, ref)
		} else {
			local = append(local, ref)
		}
	}
	return append(local, remote...), nil
}
