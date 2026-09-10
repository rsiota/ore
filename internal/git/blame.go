package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BlameLine is one row in the archaeology / blame grid.
type BlameLine struct {
	Line         int
	Hash         string
	ShortHash    string
	Author       string
	When         time.Time
	Summary      string
	Text         string
	PreviousHash string // commit before this line's change (porcelain "previous")
	PreviousPath string
}

// BlameOptions controls Blame.
type BlameOptions struct {
	Rev string // empty = HEAD
}

// Blame returns per-line blame for path at rev.
func (r *Repo) Blame(ctx context.Context, path string, opt BlameOptions) ([]BlameLine, error) {
	if path == "" {
		return nil, fmt.Errorf("path required")
	}
	rev := opt.Rev
	if rev == "" {
		rev = "HEAD"
	}
	out, err := r.run(ctx, "blame", "--porcelain", rev, "--", path)
	if err != nil {
		return nil, err
	}
	return parseBlamePorcelain(out)
}

func parseBlamePorcelain(out []byte) ([]BlameLine, error) {
	type meta struct {
		author       string
		when         time.Time
		summary      string
		previousHash string
		previousPath string
	}
	commitMeta := map[string]*meta{}
	getMeta := func(h string) *meta {
		if m, ok := commitMeta[h]; ok {
			return m
		}
		m := &meta{}
		commitMeta[h] = m
		return m
	}

	var (
		result  []BlameLine
		curHash string
		curLine int
		haveHdr bool
	)

	for _, raw := range strings.Split(string(out), "\n") {
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "\t") {
			if !haveHdr {
				continue
			}
			m := getMeta(curHash)
			short := curHash
			if len(short) > 7 {
				short = short[:7]
			}
			result = append(result, BlameLine{
				Line:         curLine,
				Hash:         curHash,
				ShortHash:    short,
				Author:       m.author,
				When:         m.when,
				Summary:      m.summary,
				Text:         raw[1:],
				PreviousHash: m.previousHash,
				PreviousPath: m.previousPath,
			})
			haveHdr = false
			continue
		}

		fields := strings.Fields(raw)
		if len(fields) >= 3 && len(fields[0]) >= 40 && isHex(fields[0][:40]) {
			curHash = fields[0]
			n, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("blame result line: %w", err)
			}
			curLine = n
			haveHdr = true
			_ = getMeta(curHash)
			continue
		}

		if curHash == "" {
			continue
		}
		m := getMeta(curHash)
		switch {
		case strings.HasPrefix(raw, "author "):
			m.author = strings.TrimPrefix(raw, "author ")
		case strings.HasPrefix(raw, "author-time "):
			sec, err := strconv.ParseInt(strings.TrimPrefix(raw, "author-time "), 10, 64)
			if err == nil {
				m.when = time.Unix(sec, 0)
			}
		case strings.HasPrefix(raw, "summary "):
			m.summary = strings.TrimPrefix(raw, "summary ")
		case strings.HasPrefix(raw, "previous "):
			rest := strings.TrimPrefix(raw, "previous ")
			parts := strings.SplitN(rest, " ", 2)
			if len(parts) >= 1 {
				m.previousHash = parts[0]
			}
			if len(parts) >= 2 {
				m.previousPath = parts[1]
			}
		}
	}
	return result, nil
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return len(s) > 0
}

// BlameAtLine returns blame for a single line (for follow probes).
func (r *Repo) BlameAtLine(ctx context.Context, path, rev string, line int) (BlameLine, error) {
	var zero BlameLine
	if line < 1 {
		return zero, fmt.Errorf("line must be >= 1")
	}
	if rev == "" {
		rev = "HEAD"
	}
	spec := fmt.Sprintf("%d,%d", line, line)
	out, err := r.run(ctx, "blame", "--porcelain", "-L", spec, rev, "--", path)
	if err != nil {
		return zero, err
	}
	lines, err := parseBlamePorcelain(out)
	if err != nil {
		return zero, err
	}
	if len(lines) == 0 {
		return zero, fmt.Errorf("no blame for %s:%d @ %s", path, line, rev)
	}
	return lines[0], nil
}
