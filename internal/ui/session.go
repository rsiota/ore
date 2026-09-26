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
		st.Main = "evolve"
		st.Path = m.evoOriginPath
		st.BlameRev = m.evoOriginRev
		st.Commit = m.filesCommitHash
		if st.Commit == "" {
			st.Commit = m.evoOriginRev
		}
		if len(m.evo) > 0 {
			st.BlameLine = m.evo[0].Line.Line
		}
		st.EvoStep = m.evoCursor
		if m.blameFrom == MainHistory {
			st.BlameFrom = "history"
		} else {
			st.BlameFrom = "files"
		}
	case MainPickaxe:
		if h, ok := m.selectedPickaxeHit(); ok {
			st.Commit = h.Commit.Hash
		}
		switch m.pickKind {
		case hitListCouple:
			st.Main = "couple"
			st.CoupleSeeds = append([]string(nil), m.coupleSeeds...)
			st.CoupleWith = m.couplePartner
			st.Path = m.couplePartner
		case hitListAuthors:
			st.Main = "authors"
			st.Author = m.authorFilter
			st.AuthorPaths = append([]string(nil), m.authorPaths...)
			st.AuthorRev = m.authorRev
			if len(m.authorPaths) == 1 {
				st.Path = m.authorPaths[0]
			}
		default:
			st.Main = "pickaxe"
			st.PickQuery = m.pickQuery
			st.PickMode = pickaxeModeLabel(m.pickMode)
			st.PickPath = m.pickPath
			st.Path = m.pickPath
		}
	}
	if m.hunkMode {
		st.Hunks = true
	}
	if n := m.effectiveLogLimit(); n > git.DefaultLogLimit {
		st.LogLimit = n
	}
	if n := m.effectiveHistoryLimit(); n > git.DefaultLogLimit {
		st.HistoryLimit = n
	}
	if n := m.effectiveHitLimit(); n > m.defaultHitLimit() {
		st.HitLimit = n
	}
	if m.mainPaneSplitW > 0 {
		st.MainPaneWidth = m.mainPaneSplitW
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
		m.restoreAfterPick = false
		m.restoreAfterEvo = false
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
	switch strings.ToLower(st.Main) {
	case "blame", "evolve":
		if st.BlameLine > 0 {
			parts = append(parts, fmt.Sprintf("L%d", st.BlameLine))
		}
	case "pickaxe":
		if st.PickQuery != "" {
			parts = append(parts, st.PickQuery)
		}
	case "couple":
		if st.CoupleWith != "" {
			parts = append(parts, st.CoupleWith)
		}
	case "authors":
		if st.Author != "" {
			parts = append(parts, st.Author)
		}
	}
	if st.Hunks {
		parts = append(parts, "hunks")
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
	m.hunkMode = st.Hunks
	if st.LogLimit > 0 {
		m.logLimit = st.LogLimit
	}
	if st.HistoryLimit > 0 {
		m.historyLimit = st.HistoryLimit
	}
	if st.HitLimit > 0 {
		m.pickLimit = st.HitLimit
	}
	if st.MainPaneWidth > 0 {
		m.mainPaneSplitW = st.MainPaneWidth
	}
	if (strings.EqualFold(st.Main, "blame") || strings.EqualFold(st.Main, "evolve")) &&
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
		return m.requestHistory(st.Path)
	case "blame", "evolve":
		if st.Path == "" {
			m.clearSessionRestore()
			return m.reloadDetail()
		}
		if strings.EqualFold(st.BlameFrom, "history") {
			m.restoreAfterHistory = true
			return m.requestHistory(st.Path)
		}
		// files → blame stack (evolve continues after blame loads)
		m.restoreAfterFiles = true
		m.openFilesPending = true
		m.detailFilterPath = ""
		m.loadingDetail = true
		return m.reloadDetailNow()
	case "pickaxe", "couple", "authors":
		return m.startRestorePickaxe(*st)
	default:
		m.clearSessionRestore()
		m.refreshStatus()
		return m.reloadDetail()
	}
}

func (m *Model) startRestorePickaxe(st session.State) tea.Cmd {
	m.main = MainCommits
	m.restoreAfterPick = true
	var next tea.Model
	var cmd tea.Cmd
	switch strings.ToLower(st.Main) {
	case "couple":
		next, cmd = m.startCouple(st.CoupleSeeds, st.CoupleWith)
	case "authors":
		rev := st.AuthorRev
		if rev == "" {
			rev = m.viewRev
		}
		next, cmd = m.startAuthors(st.Author, st.AuthorPaths, rev)
	default:
		mode := git.PickaxeString
		if strings.EqualFold(st.PickMode, "regexp") {
			mode = git.PickaxeRegexp
		}
		next, cmd = m.startPickaxe(st.PickQuery, mode, st.PickPath)
	}
	*m = next.(Model)
	if cmd == nil {
		m.clearSessionRestore()
	}
	return cmd
}

func (m *Model) finishRestoreFiles() tea.Cmd {
	st := m.pendingRestore
	if st == nil {
		return m.reloadDetail()
	}
	m.selectFilePath(st.Path)
	m.restoreAfterFiles = false
	wantBlame := strings.EqualFold(st.Main, "blame") || strings.EqualFold(st.Main, "evolve")
	if wantBlame && !strings.EqualFold(st.BlameFrom, "history") {
		rev := st.BlameRev
		if rev == "" {
			rev = m.filesCommitHash
		}
		if rev == "" {
			rev = st.Commit
		}
		if strings.EqualFold(st.Main, "evolve") {
			m.restoreAfterEvo = true
		} else {
			m.clearSessionRestore()
			m.status = fmt.Sprintf("restored · blame %s", st.Path)
		}
		cmd := m.startBlame(st.Path, rev, MainFiles)
		m.blamePreferLine = st.BlameLine
		return cmd
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
	if strings.EqualFold(st.Main, "blame") || strings.EqualFold(st.Main, "evolve") {
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
		if strings.EqualFold(st.Main, "evolve") {
			m.restoreAfterEvo = true
		} else {
			m.clearSessionRestore()
			m.status = fmt.Sprintf("restored · blame %s", path)
		}
		cmd := m.startBlame(path, rev, MainHistory)
		m.blamePreferLine = st.BlameLine
		return cmd
	}
	m.clearSessionRestore()
	m.refreshStatus()
	return m.reloadDetail()
}

func (m *Model) clearSessionRestore() {
	m.pendingRestore = nil
	m.restoreAfterFiles = false
	m.restoreAfterHistory = false
	m.restoreAfterPick = false
	m.restoreAfterEvo = false
}

func (m *Model) finishRestorePick() {
	st := m.pendingRestore
	if st == nil {
		m.restoreAfterPick = false
		return
	}
	m.selectPickaxeCommit(st.Commit)
	m.status = "restored · " + sessionSummary(*st)
	m.clearSessionRestore()
}

func (m *Model) finishRestoreEvolve() {
	st := m.pendingRestore
	if st == nil {
		m.restoreAfterEvo = false
		return
	}
	if st.EvoStep > 0 && st.EvoStep < len(m.evo) {
		m.evoCursor = st.EvoStep
		m.ensureEvoVisible()
	}
	m.status = "restored · " + sessionSummary(*st)
	m.clearSessionRestore()
}

func (m *Model) selectPickaxeCommit(hash string) {
	if hash == "" {
		return
	}
	for i, h := range m.pickaxe {
		if hashMatch(h.Commit.Hash, hash) {
			m.pickCursor = i
			m.ensurePickVisible()
			return
		}
	}
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
