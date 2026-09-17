package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// PickaxeMode selects string (-S) or regexp (-G) pickaxe search.
type PickaxeMode int

const (
	PickaxeString PickaxeMode = iota // git log -S
	PickaxeRegexp                    // git log -G
)

func (m PickaxeMode) Flag() string {
	switch m {
	case PickaxeRegexp:
		return "-G"
	default:
		return "-S"
	}
}

func (m PickaxeMode) Label() string {
	switch m {
	case PickaxeRegexp:
		return "regexp"
	default:
		return "string"
	}
}

// PickaxeOptions controls Pickaxe.
type PickaxeOptions struct {
	Query    string
	Mode     PickaxeMode
	Path     string // optional path limiter
	Rev      string // empty = HEAD
	MaxCount int    // 0 = default cap (smaller than full log)
}

// PickaxeHit is one commit that introduced or removed the pickaxe query,
// with the paths touched in that commit (when available).
type PickaxeHit struct {
	Commit Commit
	Paths  []string
}

const defaultPickaxeLimit = 100

// Pickaxe runs git log -S/-G and returns matching commits newest-first.
func (r *Repo) Pickaxe(ctx context.Context, opt PickaxeOptions) ([]PickaxeHit, error) {
	q := strings.TrimSpace(opt.Query)
	if q == "" {
		return nil, fmt.Errorf("pickaxe query required")
	}
	limit := opt.MaxCount
	if limit <= 0 {
		limit = defaultPickaxeLimit
	}
	rev := opt.Rev
	if rev == "" {
		rev = "HEAD"
	}

	format := recSep + strings.Join([]string{
		"%H", "%h", "%an", "%ae", "%aI", "%s", "%P",
	}, fieldSep)

	args := []string{
		"log", rev,
		opt.Mode.Flag() + q,
		"--name-only",
		"--date=iso-strict",
		"--pretty=format:" + format,
		"-n", strconv.Itoa(limit),
	}
	if opt.Path != "" {
		args = append(args, "--", opt.Path)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parsePickaxeLog(out)
}

func parsePickaxeLog(out []byte) ([]PickaxeHit, error) {
	raw := string(out)
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var hits []PickaxeHit
	for _, block := range strings.Split(raw, recSep) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		meta := strings.TrimSpace(lines[0])
		if meta == "" {
			continue
		}
		commits, err := parseCommitLog([]byte(meta + recSep))
		if err != nil {
			return nil, err
		}
		if len(commits) == 0 {
			continue
		}
		var paths []string
		seen := map[string]struct{}{}
		for _, line := range lines[1:] {
			p := strings.TrimSpace(line)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			paths = append(paths, p)
		}
		hits = append(hits, PickaxeHit{Commit: commits[0], Paths: paths})
	}
	return hits, nil
}
