// Package session persists per-repo archaeology workspace state so reopening
// a repository restores view tip, main pane, commit, path, and blame line.
package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// State is the persisted workspace for one repository.
type State struct {
	Version int `json:"version"`

	// ViewRev is the read-only log tip (branch/ref); empty = worktree HEAD.
	ViewRev string `json:"view_rev,omitempty"`

	// Main is commits | files | history | blame.
	Main string `json:"main,omitempty"`

	// Commit is the selected commit hash (files parent / history row / log cursor).
	Commit string `json:"commit,omitempty"`

	// Path is the selected file (files / history / blame).
	Path string `json:"path,omitempty"`

	// BlameRev / BlameLine locate blame when Main == "blame".
	BlameRev  string `json:"blame_rev,omitempty"`
	BlameLine int    `json:"blame_line,omitempty"`

	// BlameFrom is "files" or "history" (esc stack).
	BlameFrom string `json:"blame_from,omitempty"`

	DiffMode        string `json:"diff_mode,omitempty"` // zen | unified
	ZenContext      int    `json:"zen_context,omitempty"`
	DetailWrap      bool   `json:"detail_wrap,omitempty"`
	BlameGutterFold int    `json:"blame_gutter_fold,omitempty"`
}

// HasContent reports whether s is worth restoring.
func (s State) HasContent() bool {
	return s.Main != "" || s.Commit != "" || s.Path != "" || s.ViewRev != ""
}

// Store persists State per repository path as JSON under <configDir>/sessions/.
type Store struct {
	mu    sync.Mutex
	dir   string
	cache map[string]State
}

// NewStore creates a session store rooted at configDir.
func NewStore(configDir string) *Store {
	return &Store{
		dir:   filepath.Join(configDir, "sessions"),
		cache: make(map[string]State),
	}
}

// Save persists workspace state for repoPath.
func (s *Store) Save(repoPath string, st State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	st.Version = 1
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	key := cacheKey(repoPath)
	if err := os.WriteFile(s.pathFor(repoPath), data, 0o600); err != nil {
		return err
	}
	s.cache[key] = st
	return nil
}

// Load returns workspace state for repoPath. Missing file → empty State.
func (s *Store) Load(repoPath string) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := cacheKey(repoPath)
	if cached, ok := s.cache[key]; ok {
		return cached, nil
	}
	data, err := os.ReadFile(s.pathFor(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, err
	}
	s.cache[key] = st
	return st, nil
}

// Clear removes persisted state for repoPath.
func (s *Store) Clear(repoPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, cacheKey(repoPath))
	if err := os.Remove(s.pathFor(repoPath)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func cacheKey(repoPath string) string {
	return filepath.Clean(repoPath)
}

func (s *Store) pathFor(repoPath string) string {
	return filepath.Join(s.dir, fileKey(repoPath)+".json")
}

// fileKey builds a stable, filesystem-safe name from the absolute repo path.
func fileKey(repoPath string) string {
	clean := filepath.Clean(repoPath)
	sum := sha256.Sum256([]byte(clean))
	base := sanitize(filepath.Base(clean))
	return base + "_" + hex.EncodeToString(sum[:6])
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "repo"
	}
	return b.String()
}
