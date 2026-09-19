package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
	"github.com/rsiota/ore/internal/session"
)

// snapshotSession captures restore-worthy workspace fields for the open repo.
func (m Model) snapshotSession() session.State {
	st := session.State{
		ViewRev:         m.viewRev,
		Commit:          m.sessionCommitHash(),
		DiffMode:        m.diffMode.Label(),
		ZenContext:      m.zenContext,
		DetailWrap:      m.detailWrap,
		BlameGutterFold: m.blameGutterFold,
	}
	switch m.main {
	case MainCommits:
		st.Main = "commits"
	case MainFiles:
		st.Main = "files"
		st.Path = m.selectedFilePath()
		st.Commit = m.filesCommitHash
	case MainHistory:
		st.Main = "history"
		st.Path = m.historyPath
		st.Commit = m.selectedHistoryHash()
	case MainBlame:
		st.Main = "blame"
		st.Path = m.blamePath
		st.BlameRev = m.blameRev
		st.Commit = m.filesCommitHash
		if st.Commit == "" {
			st.Commit = m.blameRev
		}
		idx := m.blameIndices()
		if m.blameCursor >= 0 && m.blameCursor < len(idx) {
			st.BlameLine = m.blame[idx[m.blameCursor]].Line
		}
		if m.blameFrom == MainHistory {
			st.BlameFrom = "history"
		} else {
			st.BlameFrom = "files"
		}
	case MainLineEvo:
		// Persist as blame at the evolution origin; reopen restores the file view.
		st.Main = "blame"
		st.Path = m.evoOriginPath
		st.BlameRev = m.evoOriginRev
		st.Commit = m.filesCommitHash
		if st.Commit == "" {
			st.Commit = m.evoOriginRev
		}
		if len(m.evo) > 0 {
			st.BlameLine = m.evo[0].Line.Line
		}
		st.BlameFrom = "files"
	case MainPickaxe:
		st.Main = "commits"
		if h, ok := m.selectedPickaxeHit(); ok {
			st.Commit = h.Commit.Hash
		}
	}
	return st
}

func (m Model) sessionCommitHash() string {
	switch m.main {
	case MainFiles:
		return m.filesCommitHash
	case MainHistory:
		return m.selectedHistoryHash()
	case MainBlame:
		if m.filesCommitHash != "" {
			return m.filesCommitHash
		}
		return m.blameRev
	default:
		return m.selectedHash()
	}
}

func (m Model) selectedFilePath() string {
	idx := m.fileIndices()
	if m.fileCursor >= 0 && m.fileCursor < len(idx) {
		return m.files[idx[m.fileCursor]].Path
	}
	return ""
}

func (m Model) selectedHistoryHash() string {
	if pc, ok := m.selectedHistory(); ok {
		return pc.Hash
	}
	return ""
}

func (m Model) selectedHistory() (git.PathCommit, bool) {
	idx := m.historyIndices()
	if m.historyCursor >= 0 && m.historyCursor < len(idx) {
		return m.history[idx[m.historyCursor]], true
	}
	return git.PathCommit{}, false
}

func (m Model) selectedBlameLine() (git.BlameLine, bool) {
	idx := m.blameIndices()
	if m.blameCursor >= 0 && m.blameCursor < len(idx) {
		return m.blame[idx[m.blameCursor]], true
	}
	return git.BlameLine{}, false
}

func (m *Model) saveSession() {
	if m.sessionStore == nil || m.repo == nil {
		return
	}
	_ = m.sessionStore.Save(m.repo.Path, m.snapshotSession())
}

func (m *Model) beginQuit() tea.Cmd {
	m.saveSession()
	return tea.Quit
}

func (m *Model) exSession(args []string) tea.Cmd {
	if m.sessionStore == nil || m.repo == nil {
		m.status = "session unavailable"
		return nil
	}
	if len(args) == 0 {
		st, err := m.sessionStore.Load(m.repo.Path)
		if err != nil {
			m.status = "session: " + err.Error()
			return nil
		}
		if !st.HasContent() {
			m.status = "no saved session — :session save"
			return nil
		}
		m.status = fmt.Sprintf("session: %s", sessionSummary(st))
		return nil
	}
	switch strings.ToLower(args[0]) {
	case "save":
		m.saveSession()
		m.status = "session saved"
		return nil
	case "clear":
		if err := m.sessionStore.Clear(m.repo.Path); err != nil {
			m.status = "session: " + err.Error()
			return nil
		}
		m.pendingRestore = nil
		m.restoreAfterFiles = false
		m.restoreAfterHistory = false
		m.status = "session cleared"
		return nil
	default:
		m.status = "usage: :session [save|clear]"
		return nil
	}
}

func sessionSummary(st session.State) string {
	parts := []string{st.Main}
	if st.ViewRev != "" {
		parts = append(parts, "view "+st.ViewRev)
	}
	if st.Commit != "" {
		parts = append(parts, shortHash(st.Commit))
	}
	if st.Path != "" {
		parts = append(parts, st.Path)
	}
	if st.Main == "blame" && st.BlameLine > 0 {
		parts = append(parts, fmt.Sprintf("L%d", st.BlameLine))
	}
	return strings.Join(parts, " · ")
}

func (m *Model) applySessionChrome(st session.State) {
	switch strings.ToLower(st.DiffMode) {
	case "unified":
		m.diffMode = DiffUnified
	case "zen":
		m.diffMode = DiffZen
	}
	if st.ZenContext > 0 {
		m.zenContext = clampZenContext(st.ZenContext)
	}
	m.detailWrap = st.DetailWrap
	if strings.EqualFold(st.Main, "blame") &&
		st.BlameGutterFold >= 0 && st.BlameGutterFold <= blameGutterFoldMax {
		m.blameGutterFold = st.BlameGutterFold
		m.sessionKeepBlameFold = true
	}
}

// continueSessionRestore advances pending restore after the commit log loads.
func (m *Model) continueSessionRestore() tea.Cmd {
	st := m.pendingRestore
	if st == nil {
		return m.reloadDetail()
	}
	m.restoreCommitCursor(st.Commit)
	m.status = "restoring session…"

	switch strings.ToLower(st.Main) {
	case "files":
		m.restoreAfterFiles = true
		m.openFilesPending = true
		m.detailFilterPath = ""
		m.loadingDetail = true
		return m.reloadDetailNow()
	case "history":
		if st.Path == "" {
			m.clearSessionRestore()
			return m.reloadDetail()
		}
		m.restoreAfterHistory = true
		m.historyPath = st.Path
		m.historyCol = histColHash
		m.historySortCol = -1
		m.historySortDir = SortNone
		m.loadingHistory = true
		return loadHistoryCmd(m.repo, st.Path)
	case "blame":
		if st.Path == "" {
			m.clearSessionRestore()
			return m.reloadDetail()
		}
		if strings.EqualFold(st.BlameFrom, "history") {
			m.restoreAfterHistory = true
			m.historyPath = st.Path
			m.historyCol = histColHash
			m.historySortCol = -1
			m.historySortDir = SortNone
			m.loadingHistory = true
			return loadHistoryCmd(m.repo, st.Path)
		}
		// files → blame stack
		m.restoreAfterFiles = true
		m.openFilesPending = true
		m.detailFilterPath = ""
		m.loadingDetail = true
		return m.reloadDetailNow()
	default:
		m.clearSessionRestore()
		m.refreshStatus()
		return m.reloadDetail()
	}
}

func (m *Model) finishRestoreFiles() tea.Cmd {
	st := m.pendingRestore
	if st == nil {
		return m.reloadDetail()
	}
	m.selectFilePath(st.Path)
	m.restoreAfterFiles = false
	if strings.EqualFold(st.Main, "blame") && !strings.EqualFold(st.BlameFrom, "history") {
		rev := st.BlameRev
		if rev == "" {
			rev = m.filesCommitHash
		}
		if rev == "" {
			rev = st.Commit
		}
		m.blamePreferLine = st.BlameLine
		m.clearSessionRestore()
		m.status = fmt.Sprintf("restored · blame %s", st.Path)
		return m.startBlame(st.Path, rev, MainFiles)
	}
	m.clearSessionRestore()
	m.refreshStatus()
	return m.reloadDetail()
}

func (m *Model) finishRestoreHistory() tea.Cmd {
	st := m.pendingRestore
	if st == nil {
		return m.reloadDetail()
	}
	m.selectHistoryCommit(st.Commit)
	m.restoreAfterHistory = false
	if strings.EqualFold(st.Main, "blame") {
		rev := st.BlameRev
		if rev == "" {
			rev = m.selectedHistoryHash()
		}
		if rev == "" {
			rev = st.Commit
		}
		path := st.Path
		if pc, ok := m.selectedHistory(); ok && pc.Path != "" {
			path = pc.Path
		}
		m.blamePreferLine = st.BlameLine
		m.clearSessionRestore()
		m.status = fmt.Sprintf("restored · blame %s", path)
		return m.startBlame(path, rev, MainHistory)
	}
	m.clearSessionRestore()
	m.refreshStatus()
	return m.reloadDetail()
}

func (m *Model) clearSessionRestore() {
	m.pendingRestore = nil
	m.restoreAfterFiles = false
	m.restoreAfterHistory = false
}

func (m *Model) selectFilePath(path string) {
	if path == "" {
		return
	}
	idx := m.fileIndices()
	for i, src := range idx {
		if m.files[src].Path == path {
			m.fileCursor = i
			m.ensureFileVisible()
			return
		}
	}
}

func (m *Model) selectHistoryCommit(hash string) {
	if hash == "" {
		return
	}
	idx := m.historyIndices()
	for i, src := range idx {
		if hashMatch(m.history[src].Hash, hash) {
			m.historyCursor = i
			m.ensureHistoryVisible()
			return
		}
	}
}
