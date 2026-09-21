package ui

import (
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/bookmarks"
	"github.com/rsiota/ore/internal/session"
)

func (m *Model) addBookmark(name string) {
	if m.bookmarkStore == nil || m.repo == nil {
		m.status = "bookmarks unavailable"
		return
	}
	st := m.snapshotSession()
	if !st.HasContent() && st.Main == "" {
		st.Main = "commits"
	}
	b := bookmarks.Bookmark{
		Name:    strings.TrimSpace(name),
		SavedAt: time.Now(),
		View:    st,
	}
	err := m.bookmarkStore.Add(m.repo.Path, b)
	switch {
	case errors.Is(err, bookmarks.ErrDuplicate):
		m.status = "already bookmarked · " + b.Label()
	case err != nil:
		m.status = "bookmark: " + err.Error()
	default:
		m.status = "bookmarked · " + b.Label()
	}
}

func (m *Model) toggleBookmarks() {
	if m.bookmarkStore == nil || m.repo == nil {
		m.status = "bookmarks unavailable"
		return
	}
	if m.bookmarks.IsVisible() {
		m.bookmarks.Hide()
		m.refreshStatus()
		return
	}
	m.palette.Hide()
	m.branches.Hide()
	if m.focus == FocusDetail {
		m.leaveDetailYank()
	}
	entries, err := m.bookmarkStore.Get(m.repo.Path)
	if err != nil {
		m.status = "bookmarks: " + err.Error()
		return
	}
	m.bookmarks.Open(entries)
	m.status = "bookmarks"
}

func (m *Model) deleteSelectedBookmark() {
	if m.bookmarkStore == nil || m.repo == nil || !m.bookmarks.IsVisible() {
		return
	}
	entries, err := m.bookmarkStore.Get(m.repo.Path)
	if err != nil || len(entries) == 0 {
		return
	}
	idx := m.bookmarks.selectedStoreIndex(len(entries))
	if idx < 0 {
		return
	}
	label := entries[idx].Label()
	if err := m.bookmarkStore.RemoveAt(m.repo.Path, idx); err != nil {
		m.status = "bookmark: " + err.Error()
		return
	}
	entries, _ = m.bookmarkStore.Get(m.repo.Path)
	m.bookmarks.SetEntries(entries)
	m.status = "deleted · " + label
}

func (m *Model) jumpBookmark(st session.State) tea.Cmd {
	if m.focus == FocusDetail {
		m.leaveDetailYank()
	}
	if m.explorer.Opened() {
		m.explorer.Close()
	}
	m.focus = FocusMain
	m.chordG = false
	m.bookmarks.Hide()
	m.palette.Hide()
	m.branches.Hide()

	cp := st
	m.pendingRestore = &cp
	m.applySessionChrome(st)
	m.status = "jumping · " + bookmarks.ViewSummary(st)

	needCommits := true
	switch strings.ToLower(st.Main) {
	case "history":
		needCommits = false
	case "blame":
		if strings.EqualFold(st.BlameFrom, "history") {
			needCommits = false
		}
	case "commits", "":
		needCommits = false
	}
	if needCommits {
		m.main = MainCommits
	}

	if st.ViewRev != m.viewRev {
		m.viewRev = st.ViewRev
		m.refreshPreferHash = ""
		m.loading = true
		m.main = MainCommits
		m.files = nil
		m.filesCommitHash = ""
		m.openFilesPending = false
		m.history = nil
		m.historyPath = ""
		return loadCommitsCmd(m.repo, m.viewRev)
	}
	return m.continueSessionRestore()
}

func (m *Model) exBookmark(args []string) tea.Cmd {
	name := strings.Join(args, " ")
	m.addBookmark(name)
	return nil
}

func (m *Model) exBookmarks(args []string) tea.Cmd {
	if len(args) > 0 && strings.EqualFold(args[0], "clear") {
		if m.bookmarkStore == nil || m.repo == nil {
			m.status = "bookmarks unavailable"
			return nil
		}
		if err := m.bookmarkStore.Clear(m.repo.Path); err != nil {
			m.status = "bookmarks: " + err.Error()
			return nil
		}
		m.bookmarks.Hide()
		m.status = "bookmarks cleared"
		return nil
	}
	m.toggleBookmarks()
	return nil
}
