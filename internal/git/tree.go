package git

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
)

// TreeEntry is one child of a tree at a revision (file, directory, or gitlink).
type TreeEntry struct {
	Name string
	Path string // repo-relative, slash-separated
	Dir  bool
}

// ListTree lists one directory of the tree at rev (empty = HEAD).
// prefix is "" for the repository root.
func (r *Repo) ListTree(ctx context.Context, rev, prefix string) ([]TreeEntry, error) {
	if rev == "" {
		rev = "HEAD"
	}
	prefix = normalizeTreePrefix(prefix)
	args := []string{"ls-tree", "-z", rev}
	if prefix != "" {
		// Trailing slash lists the directory's children, not the tree itself.
		args = append(args, "--", prefix+"/")
	}
	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseLSTree(out, prefix), nil
}

// ListFiles lists every blob path under prefix at rev (empty rev = HEAD, empty
// prefix = whole tree). Used for sidebar path filter.
func (r *Repo) ListFiles(ctx context.Context, rev, prefix string) ([]string, error) {
	if rev == "" {
		rev = "HEAD"
	}
	prefix = normalizeTreePrefix(prefix)
	args := []string{"ls-tree", "-r", "-z", "--name-only", rev}
	if prefix != "" {
		args = append(args, "--", prefix)
	}
	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseNULNames(out), nil
}

func normalizeTreePrefix(prefix string) string {
	prefix = strings.Trim(strings.ReplaceAll(prefix, "\\", "/"), "/")
	return prefix
}

func parseLSTree(out []byte, prefix string) []TreeEntry {
	recs := bytes.Split(out, []byte{0})
	entries := make([]TreeEntry, 0, len(recs))
	for _, rec := range recs {
		if len(rec) == 0 {
			continue
		}
		tab := bytes.IndexByte(rec, '\t')
		if tab < 0 {
			continue
		}
		meta := string(rec[:tab])
		path := string(rec[tab+1:])
		path = strings.TrimPrefix(path, "./")
		if path == "" {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) < 2 {
			continue
		}
		kind := fields[1]
		if kind == "commit" {
			// Submodule gitlink — treat as a leaf.
			kind = "blob"
		}
		if kind != "tree" && kind != "blob" {
			continue
		}
		if prefix != "" {
			// ls-tree REV -- dir lists children as dir/name (or the dir itself).
			if path == prefix {
				continue
			}
			if !strings.HasPrefix(path, prefix+"/") {
				continue
			}
		}
		name := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			name = path[i+1:]
		}
		entries = append(entries, TreeEntry{
			Name: name,
			Path: path,
			Dir:  kind == "tree",
		})
	}
	sortTreeEntries(entries)
	return entries
}

func parseNULNames(out []byte) []string {
	recs := bytes.Split(out, []byte{0})
	names := make([]string, 0, len(recs))
	for _, rec := range recs {
		if len(rec) == 0 {
			continue
		}
		names = append(names, string(rec))
	}
	sort.Strings(names)
	return names
}

func sortTreeEntries(entries []TreeEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

// TreeParent returns the parent directory of a repo-relative path, or "".
func TreeParent(path string) string {
	path = normalizeTreePrefix(path)
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return ""
	}
	return path[:i]
}

// TreeBase returns the final path component.
func TreeBase(path string) string {
	path = normalizeTreePrefix(path)
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// TreeAncestors returns directory prefixes from the root down to path's parent.
func TreeAncestors(path string) []string {
	path = normalizeTreePrefix(path)
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return []string{""}
	}
	out := make([]string, 0, len(parts))
	out = append(out, "")
	cur := parts[0]
	for i := 1; i < len(parts); i++ {
		out = append(out, cur)
		if i < len(parts)-1 {
			cur = fmt.Sprintf("%s/%s", cur, parts[i])
		}
	}
	return out
}
