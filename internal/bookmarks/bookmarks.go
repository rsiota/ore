// Package bookmarks persists named archaeology views per repository.
package bookmarks

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rsiota/ore/internal/session"
)

// Bookmark is a named snapshot of an archaeology workspace location.
type Bookmark struct {
	Name    string        `json:"name,omitempty"`
	SavedAt time.Time     `json:"saved_at"`
	View    session.State `json:"view"`
}

// Label returns a display name: explicit Name, else a compact view summary.
func (b Bookmark) Label() string {
	if strings.TrimSpace(b.Name) != "" {
		return strings.TrimSpace(b.Name)
	}
	return ViewSummary(b.View)
}

// ViewSummary builds a compact one-line description of a view.
func ViewSummary(st session.State) string {
	parts := []string{}
	if st.Main != "" {
		parts = append(parts, st.Main)
	} else {
		parts = append(parts, "commits")
	}
	if st.ViewRev != "" {
		parts = append(parts, "view "+st.ViewRev)
	}
	if st.Commit != "" {
		h := st.Commit
		if len(h) > 7 {
			h = h[:7]
		}
		parts = append(parts, h)
	}
	if st.Path != "" {
		parts = append(parts, st.Path)
	}
	if strings.EqualFold(st.Main, "blame") && st.BlameLine > 0 {
		parts = append(parts, fmt.Sprintf("L%d", st.BlameLine))
	}
	return strings.Join(parts, " · ")
}

// SameView reports whether two views point at the same archaeology location
// (ignores chrome like zen context / wrap).
func SameView(a, b session.State) bool {
	return strings.EqualFold(a.Main, b.Main) &&
		a.ViewRev == b.ViewRev &&
		a.Commit == b.Commit &&
		a.Path == b.Path &&
		a.BlameRev == b.BlameRev &&
		a.BlameLine == b.BlameLine &&
		strings.EqualFold(a.BlameFrom, b.BlameFrom)
}

// Store manages bookmarks per repository under <configDir>/bookmarks/.
type Store struct {
	mu    sync.Mutex
	dir   string
	cache map[string][]Bookmark
}

// NewStore creates a bookmark store rooted at configDir.
func NewStore(configDir string) *Store {
	return &Store{
		dir:   filepath.Join(configDir, "bookmarks"),
		cache: make(map[string][]Bookmark),
	}
}

func (s *Store) pathFor(repoPath string) string {
	return filepath.Join(s.dir, fileKey(repoPath)+".json")
}

func (s *Store) load(repoPath string) ([]Bookmark, error) {
	key := cacheKey(repoPath)
	if cached, ok := s.cache[key]; ok {
		return cached, nil
	}
	data, err := os.ReadFile(s.pathFor(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return []Bookmark{}, nil
		}
		return nil, err
	}
	var entries []Bookmark
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	s.cache[key] = entries
	return entries, nil
}

func (s *Store) save(repoPath string, entries []Bookmark) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.pathFor(repoPath), data, 0o600)
}

// Add saves a bookmark for repoPath. Returns ErrDuplicate if SameView matches.
func (s *Store) Add(repoPath string, b Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.load(repoPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if SameView(e.View, b.View) {
			return ErrDuplicate
		}
	}
	if b.SavedAt.IsZero() {
		b.SavedAt = time.Now()
	}
	entries = append(entries, b)
	s.cache[cacheKey(repoPath)] = entries
	return s.save(repoPath, entries)
}

// Get returns all bookmarks for repoPath (oldest first).
func (s *Store) Get(repoPath string) ([]Bookmark, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(repoPath)
}

// RemoveAt deletes the bookmark at index (in the full list, oldest-first).
func (s *Store) RemoveAt(repoPath string, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.load(repoPath)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(entries) {
		return nil
	}
	entries = append(entries[:index], entries[index+1:]...)
	s.cache[cacheKey(repoPath)] = entries
	return s.save(repoPath, entries)
}

// Clear removes all bookmarks for repoPath.
func (s *Store) Clear(repoPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[cacheKey(repoPath)] = nil
	path := s.pathFor(repoPath)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func cacheKey(repoPath string) string {
	return filepath.Clean(repoPath)
}

// fileKey mirrors session's naming so bookmarks sit next to sessions stably.
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

// ErrDuplicate is returned when the same view is already bookmarked.
var ErrDuplicate = errDuplicate{}

type errDuplicate struct{}

func (errDuplicate) Error() string { return "already bookmarked" }

// FormatTime returns a compact timestamp for panel display.
func FormatTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}
