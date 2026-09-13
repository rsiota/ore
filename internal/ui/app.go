// Package ui is the Bubble Tea front-end for ore.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rsiota/ore/internal/git"
)

// Focus is which pane receives keys.
type Focus int

const (
	FocusMain Focus = iota
	FocusDetail
	FocusExplorer
)

// MainView is the left/main grid mode.
type MainView int

const (
	MainCommits MainView = iota
	MainFiles
	MainHistory
	MainBlame
)

// Model is the root TUI state.
type Model struct {
	repo *git.Repo

	width  int
	height int
	focus  Focus
	main   MainView

	commits         []git.Commit
	cursor          int
	commitOffset    int
	commitCol       int // active column (cursor / sort target)
	commitFilterCol int // column the active / filter applies to
	commitSortCol   int
	commitSortDir   SortDir

	files            []git.FileChange
	fileCursor       int
	fileOffset       int
	fileCol          int
	fileFilterCol    int
	fileSortCol      int
	fileSortDir      SortDir
	filesCommitHash  string
	openFilesPending bool

	historyPath      string
	history          []git.Commit
	historyCursor    int
	historyOffset    int
	historyCol       int
	historyFilterCol int
	historySortCol   int
	historySortDir   SortDir
	loadingHistory   bool

	blame           []git.BlameLine
	blamePath       string
	blameRev        string
	blameCursor     int
	blameOffset     int
	blameCol        int
	blameFilterCol  int
	blameSortCol    int
	blameSortDir    SortDir
	blameFrom       MainView // MainFiles or MainHistory
	blamePreferLine int
	loadingBlame    bool
	chordG          bool // pending g-prefix for gg / gf in blame

	detail           *git.CommitDetail
	detailFilterPath string // path requested by the in-flight / latest reloadDetail
	detailPath       string // path the current detail payload was loaded with ("" = whole commit)
	detailExpectHash string // if set, detail load may target this hash (e.g. :goto)
	detailOffset     int
	diffMode         DiffMode // zen (default) or unified patch
	zenContext       int      // git -U / in-hunk context lines
	detailWrap       bool     // soft-wrap long detail lines
	detailTruncated    bool // patch soft-capped for responsiveness
	detailPatchPending bool // header fetched; waiting to paint until patch arrives
	detailPending      *git.CommitDetail // staged header; not shown until patch lands
	detailDebounceSeq  int
	detailLoadSeq      int
	detailCancel       context.CancelFunc
	detailCtx          context.Context
	detailCache        *detailRenderCache // pointer so View (value recv) can memoize
	loading            bool
	loadingDetail      bool
	err                string
	status             string

	branch string
	head   string

	help    HelpPanel
	palette palette

	filterTyping bool
	filter       string // applied / live query

	exTyping bool
	exLine   string

	explorer   RelExplorer
	loadingRel bool

	// refreshPreferHash, when set, marks commitsLoadedMsg as a refresh rather
	// than the initial load: restore the commit cursor to this hash and keep
	// the current main view.
	refreshPreferHash string
}

// New builds a model bound to repo. Call Init via the Bubble Tea program.
func New(repo *git.Repo) Model {
	return Model{
		repo:           repo,
		focus:          FocusMain,
		main:           MainCommits,
		loading:        true,
		status:         "loading commits…",
		commitSortCol:  -1,
		commitCol:      commitColHash,
		fileSortCol:    -1,
		historySortCol: -1,
		blameSortCol:   -1,
		diffMode:       DiffZen,
		zenContext:     defaultZenContext,
		detailCache:    &detailRenderCache{},
	}
}

type commitsLoadedMsg struct {
	commits []git.Commit
	branch  string
	head    string
	err     error
}

type historyLoadedMsg struct {
	path    string
	commits []git.Commit
	err     error
}

type blameLoadedMsg struct {
	path  string
	rev   string
	lines []git.BlameLine
	err   error
}

type relationsLoadedMsg struct {
	commit *git.CommitRelations
	line   *git.LineRelations
	err    error
}

func loadCommitsCmd(repo *git.Repo) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		commits, err := repo.CommitLog(ctx, git.LogOptions{})
		return commitsLoadedMsg{
			commits: commits,
			branch:  repo.BranchName(ctx),
			head:    repo.HeadShort(ctx),
			err:     err,
		}
	}
}

func loadHistoryCmd(repo *git.Repo, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		commits, err := repo.FileHistory(ctx, path, git.LogOptions{})
		return historyLoadedMsg{path: path, commits: commits, err: err}
	}
}

func loadBlameCmd(repo *git.Repo, path, rev string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		lines, err := repo.Blame(ctx, path, git.BlameOptions{Rev: rev})
		return blameLoadedMsg{path: path, rev: rev, lines: lines, err: err}
	}
}

func loadCommitRelationsCmd(repo *git.Repo, hash string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		rel, err := repo.Relations(ctx, hash)
		if err != nil {
			return relationsLoadedMsg{err: err}
		}
		return relationsLoadedMsg{commit: &rel}
	}
}

func loadLineRelationsCmd(repo *git.Repo, path, rev string, line git.BlameLine) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		rel, err := repo.LineRelationsAt(ctx, path, rev, line)
		if err != nil {
			return relationsLoadedMsg{err: err}
		}
		return relationsLoadedMsg{line: &rel}
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return loadCommitsCmd(m.repo)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetSize(msg.Width, msg.Height)
		m.layoutExplorer()
		return m, nil

	case commitsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		prefer := m.refreshPreferHash
		m.refreshPreferHash = ""
		m.commits = msg.commits
		m.branch = msg.branch
		m.head = msg.head
		if prefer != "" {
			m.restoreCommitCursor(prefer)
			m.status = fmt.Sprintf("refreshed · %d commits", len(m.commits))
			return m, m.afterRefreshCmds()
		}
		m.cursor = 0
		m.commitOffset = 0
		m.main = MainCommits
		m.status = fmt.Sprintf("%d commits", len(m.commits))
		if len(m.commits) > 0 {
			return m, m.reloadDetail()
		}
		return m, nil

	case detailDebounceMsg:
		return m.handleDetailDebounce(msg)

	case detailHeaderMsg:
		return m.handleDetailHeader(msg)

	case detailPatchMsg:
		return m.handleDetailPatch(msg)

	case historyLoadedMsg:
		m.loadingHistory = false
		if msg.path != m.historyPath {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.history = msg.commits
		m.historyCursor = 0
		m.historyOffset = 0
		m.historyCol = commitColHash
		m.historySortCol = -1
		m.historySortDir = SortNone
		m.filter = ""
		m.filterTyping = false
		m.main = MainHistory
		m.focus = FocusMain
		m.status = fmt.Sprintf("history · %s · %d commits · b/enter blame", m.historyPath, len(m.history))
		if len(m.history) > 0 {
			return m, m.reloadDetail()
		}
		m.detail = nil
		m.detailPath = ""
		return m, nil

	case blameLoadedMsg:
		m.loadingBlame = false
		if msg.path != m.blamePath || msg.rev != m.blameRev {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.blame = msg.lines
		m.blameCursor = 0
		m.blameOffset = 0
		m.blameCol = 0
		m.blameSortCol = -1
		m.blameSortDir = SortNone
		if m.blamePreferLine > 0 {
			for i, bl := range m.blame {
				if bl.Line >= m.blamePreferLine {
					m.blameCursor = i
					break
				}
				m.blameCursor = i
			}
			m.blamePreferLine = 0
			m.ensureBlameVisible()
		}
		m.main = MainBlame
		m.focus = FocusMain
		m.chordG = false
		m.status = fmt.Sprintf("blame · %s @ %s · %d lines · f follow · esc back",
			m.blamePath, shortHash(m.blameRev), len(m.blame))
		if len(m.blame) > 0 {
			return m, m.reloadDetail()
		}
		m.detail = nil
		m.detailPath = ""
		return m, nil

	case relationsLoadedMsg:
		m.loadingRel = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.err = ""
		if msg.line != nil {
			m.explorer.LoadLine(*msg.line)
		} else if msg.commit != nil {
			m.explorer.LoadCommit(*msg.commit)
		}
		m.layoutExplorer()
		m.focus = FocusExplorer
		m.status = "relationships · enter open · esc close"
		return m, nil

	case paletteExMsg:
		if msg.typing {
			m.beginEx()
			m.exLine = msg.line
			m.status = ":" + m.exLine + "█"
			return m, nil
		}
		return m, m.runExCommand(msg.line)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) selectedHash() string {
	switch m.main {
	case MainBlame:
		idx := m.blameIndices()
		if m.blameCursor < 0 || m.blameCursor >= len(idx) {
			return ""
		}
		return m.blame[idx[m.blameCursor]].Hash
	case MainHistory:
		idx := m.historyIndices()
		if m.historyCursor < 0 || m.historyCursor >= len(idx) {
			return ""
		}
		return m.history[idx[m.historyCursor]].Hash
	case MainFiles:
		return m.filesCommitHash
	default:
		idx := m.commitIndices()
		if m.cursor < 0 || m.cursor >= len(idx) {
			return ""
		}
		return m.commits[idx[m.cursor]].Hash
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.palette.IsVisible() {
		var cmd tea.Cmd
		m.palette, cmd = m.palette.Update(msg)
		return m, cmd
	}

	if m.help.Visible() {
		if msg.String() == "ctrl+p" {
			m.help.Hide()
			m.palette.Open()
			m.status = "command palette"
			return m, nil
		}
		m.help.Update(msg)
		return m, nil
	}

	if m.exTyping {
		return m.handleExKeys(msg)
	}

	if m.filterTyping {
		return m.handleFilterKeys(msg)
	}

	if m.focus == FocusExplorer && m.explorer.Opened() {
		return m.handleExplorerKeys(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "ctrl+p":
		m.chordG = false
		m.palette.Open()
		m.status = "command palette"
		return m, nil
	case "D":
		m.chordG = false
		m.diffMode = m.diffMode.Next()
		m.detailOffset = 0
		m.invalidateDetailCache()
		m.status = "diff " + m.diffMode.Label()
		return m, nil
	case "w":
		m.chordG = false
		m.detailWrap = !m.detailWrap
		m.detailOffset = 0
		m.invalidateDetailCache()
		if m.detailWrap {
			m.status = "diff wrap on"
		} else {
			m.status = "diff wrap off"
		}
		return m, nil
	case "[":
		if m.diffMode != DiffZen {
			break
		}
		m.chordG = false
		next := clampZenContext(m.zenContext - 1)
		if next == m.zenContext {
			m.status = fmt.Sprintf("diff context %d", m.zenContext)
			return m, nil
		}
		m.zenContext = next
		m.detailOffset = 0
		m.status = fmt.Sprintf("diff context %d", m.zenContext)
		return m, m.reloadDetail()
	case "]":
		if m.diffMode != DiffZen {
			break
		}
		m.chordG = false
		next := clampZenContext(m.zenContext + 1)
		if next == m.zenContext {
			m.status = fmt.Sprintf("diff context %d", m.zenContext)
			return m, nil
		}
		m.zenContext = next
		m.detailOffset = 0
		m.status = fmt.Sprintf("diff context %d", m.zenContext)
		return m, m.reloadDetail()
	case "ctrl+r":
		m.chordG = false
		return m, m.refresh()
	case "tab":
		m.chordG = false
		return m.cycleFocus()
	case "esc", "backspace":
		m.chordG = false
		if m.explorer.Opened() && msg.String() == "esc" {
			m.explorer.Close()
			m.focus = FocusMain
			m.refreshStatus()
			return m, m.reloadDetail()
		}
		if m.filter != "" && msg.String() == "esc" {
			m.clearFilter()
			return m, nil
		}
		return m.goBack()
	case ":":
		m.beginEx()
		m.status = ":█"
		return m, nil
	case "/":
		if m.focus == FocusMain {
			m.chordG = false
			m.filterTyping = true
			switch m.main {
			case MainCommits:
				m.commitFilterCol = m.commitCol
			case MainFiles:
				m.fileFilterCol = m.fileCol
			case MainHistory:
				m.historyFilterCol = m.historyCol
			case MainBlame:
				m.blameFilterCol = m.blameCol
			}
			m.status = m.filterPrompt()
			return m, nil
		}
	case "?":
		m.chordG = false
		m.help.Toggle()
		return m, nil
	}

	switch m.focus {
	case FocusMain:
		return m.handleMainKeys(msg)
	case FocusDetail:
		return m.handleDetailKeys(msg)
	}
	return m, nil
}

func (m Model) cycleFocus() (tea.Model, tea.Cmd) {
	if m.explorer.Opened() {
		switch m.focus {
		case FocusMain:
			m.focus = FocusExplorer
		case FocusExplorer:
			m.focus = FocusMain
		default:
			m.focus = FocusMain
		}
		return m, nil
	}
	if m.focus == FocusMain {
		m.focus = FocusDetail
	} else {
		m.focus = FocusMain
	}
	return m, nil
}

func (m Model) handleExplorerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	consumed, activate := m.explorer.Update(msg)
	if !consumed {
		return m, nil
	}
	if !m.explorer.Opened() {
		m.focus = FocusMain
		m.refreshStatus()
		return m, m.reloadDetail()
	}
	if activate {
		return m.activateRelation()
	}
	return m, nil
}

func (m Model) openRelations() (tea.Model, tea.Cmd) {
	m.chordG = false
	m.loadingRel = true
	m.status = "loading relationships…"
	if m.main == MainBlame {
		idx := m.blameIndices()
		if m.blameCursor < 0 || m.blameCursor >= len(idx) {
			m.loadingRel = false
			m.status = "no blame line selected"
			return m, nil
		}
		line := m.blame[idx[m.blameCursor]]
		return m, loadLineRelationsCmd(m.repo, m.blamePath, m.blameRev, line)
	}
	hash := m.selectedHash()
	if hash == "" {
		m.loadingRel = false
		m.status = "no commit selected"
		return m, nil
	}
	return m, loadCommitRelationsCmd(m.repo, hash)
}

func (m Model) activateRelation() (tea.Model, tea.Cmd) {
	row, ok := m.explorer.Selected()
	if !ok {
		return m, nil
	}
	switch row.kind {
	case relCommit:
		m.explorer.Close()
		m.focus = FocusMain
		// Jump main grid to this commit when possible.
		if m.jumpToCommit(row.hash) {
			m.refreshStatus()
			return m, m.reloadDetail()
		}
		// Commit not in current log window — still show detail.
		m.detailFilterPath = ""
		m.detailExpectHash = row.hash
		m.loadingDetail = true
		m.main = MainCommits
		return m, m.reloadDetailNow()
	case relFile:
		path := row.path
		m.explorer.Close()
		m.focus = FocusMain
		m.historyPath = path
		m.loadingHistory = true
		m.status = fmt.Sprintf("loading history · %s", path)
		return m, loadHistoryCmd(m.repo, path)
	default:
		return m, nil
	}
}

func (m *Model) jumpToCommit(hash string) bool {
	for i, c := range m.commits {
		if c.Hash == hash || strings.HasPrefix(c.Hash, hash) || strings.HasPrefix(hash, c.Hash) {
			m.main = MainCommits
			m.filter = ""
			m.filterTyping = false
			// cursor indexes filtered list; with empty filter == raw index
			m.cursor = i
			m.ensureCommitVisible()
			return true
		}
	}
	return false
}

func (m *Model) layoutExplorer() {
	h := m.paneContentHeight()
	if m.width < 80 {
		m.explorer.SetSize(max(1, m.width-borderOverhead), h)
		return
	}
	cw := m.mainPaneWidth()
	dw := m.width - cw
	m.explorer.SetSize(max(1, dw-borderOverhead), h)
}

func (m Model) filterPrompt() string {
	switch m.main {
	case MainCommits:
		col := m.commitFilterCol
		if col >= 0 && col < len(commitColumns) {
			return "filter " + commitColumns[col] + ": " + m.filter + "█"
		}
	case MainFiles:
		col := m.fileFilterCol
		if col >= 0 && col < len(fileColumns) {
			return "filter " + fileColumns[col] + ": " + m.filter + "█"
		}
	case MainHistory:
		col := m.historyFilterCol
		if col >= 0 && col < len(commitColumns) {
			return "filter " + commitColumns[col] + ": " + m.filter + "█"
		}
	case MainBlame:
		col := m.blameFilterCol
		if col >= 0 && col < len(blameColumns) {
			return "filter " + blameColumns[col] + ": " + m.filter + "█"
		}
	}
	return "filter: " + m.filter + "█"
}

func (m Model) handleFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.clearFilter()
		return m, nil
	case "enter":
		m.filterTyping = false
		m.refreshStatus()
		return m, nil
	case "backspace":
		if m.filter != "" {
			r := []rune(m.filter)
			m.filter = string(r[:len(r)-1])
			m.clampMainCursor()
		}
		m.status = m.filterPrompt()
		return m, nil
	case "ctrl+u":
		m.filter = ""
		m.clampMainCursor()
		m.status = m.filterPrompt()
		return m, nil
	default:
		if len(msg.Runes) > 0 && !msg.Alt && msg.Type == tea.KeyRunes {
			m.filter += string(msg.Runes)
			m.clampMainCursor()
			m.status = m.filterPrompt()
			return m, nil
		}
		if s := msg.String(); len(s) == 1 && s[0] >= 32 {
			m.filter += s
			m.clampMainCursor()
			m.status = m.filterPrompt()
		}
	}
	return m, nil
}

func (m *Model) clearFilter() {
	m.filter = ""
	m.filterTyping = false
	m.commitFilterCol = 0
	m.clampMainCursor()
	m.refreshStatus()
}

func (m *Model) refreshStatus() {
	if m.err != "" {
		return
	}
	switch m.main {
	case MainCommits:
		m.status = fmt.Sprintf("%d commits", len(m.commitIndices()))
		if m.commitSortDir != SortNone && m.commitSortCol >= 0 && m.commitSortCol < len(commitColumns) {
			m.status += fmt.Sprintf(" · sort %s%s", commitColumns[m.commitSortCol], m.commitSortDir.Arrow())
		}
		if m.filter != "" {
			col := ""
			if m.commitFilterCol >= 0 && m.commitFilterCol < len(commitColumns) {
				col = commitColumns[m.commitFilterCol] + " "
			}
			m.status += " · /" + col + m.filter
		}
	case MainFiles:
		m.status = fmt.Sprintf("%d files in %s", len(m.fileIndices()), shortHash(m.filesCommitHash))
		if m.fileSortDir != SortNone && m.fileSortCol >= 0 && m.fileSortCol < len(fileColumns) {
			m.status += fmt.Sprintf(" · sort %s%s", fileColumns[m.fileSortCol], m.fileSortDir.Arrow())
		}
		if m.filter != "" {
			col := ""
			if m.fileFilterCol >= 0 && m.fileFilterCol < len(fileColumns) {
				col = fileColumns[m.fileFilterCol] + " "
			}
			m.status += " · /" + col + m.filter
		}
	case MainHistory:
		m.status = fmt.Sprintf("history · %s · %d commits", m.historyPath, len(m.historyIndices()))
		if m.historySortDir != SortNone && m.historySortCol >= 0 && m.historySortCol < len(commitColumns) {
			m.status += fmt.Sprintf(" · sort %s%s", commitColumns[m.historySortCol], m.historySortDir.Arrow())
		}
		if m.filter != "" {
			col := ""
			if m.historyFilterCol >= 0 && m.historyFilterCol < len(commitColumns) {
				col = commitColumns[m.historyFilterCol] + " "
			}
			m.status += " · /" + col + m.filter
		}
	case MainBlame:
		m.status = fmt.Sprintf("blame · %s @ %s · %d lines", m.blamePath, shortHash(m.blameRev), len(m.blameIndices()))
		if m.blameSortDir != SortNone && m.blameSortCol >= 0 && m.blameSortCol < len(blameColumns) {
			m.status += fmt.Sprintf(" · sort %s%s", blameColumns[m.blameSortCol], m.blameSortDir.Arrow())
		}
		if m.filter != "" {
			col := ""
			if m.blameFilterCol >= 0 && m.blameFilterCol < len(blameColumns) {
				col = blameColumns[m.blameFilterCol] + " "
			}
			m.status += " · /" + col + m.filter
		}
	}
}

func (m Model) goBack() (tea.Model, tea.Cmd) {
	switch m.main {
	case MainBlame:
		m.blame = nil
		m.blamePath = ""
		m.blameRev = ""
		m.chordG = false
		m.focus = FocusMain
		if m.blameFrom == MainHistory && m.historyPath != "" {
			m.main = MainHistory
			m.status = fmt.Sprintf("history · %s · %d commits", m.historyPath, len(m.history))
			return m, m.reloadDetail()
		}
		m.main = MainFiles
		if len(m.files) > 0 {
			m.status = fmt.Sprintf("%d files in %s", len(m.files), shortHash(m.filesCommitHash))
			return m, m.reloadDetail()
		}
		m.main = MainCommits
		m.status = fmt.Sprintf("%d commits", len(m.commits))
		return m, m.reloadDetail()
	case MainHistory:
		m.main = MainFiles
		m.history = nil
		m.historyPath = ""
		m.focus = FocusMain
		if len(m.files) > 0 {
			m.status = fmt.Sprintf("%d files in %s", len(m.files), shortHash(m.filesCommitHash))
			return m, m.reloadDetail()
		}
		m.main = MainCommits
		m.status = fmt.Sprintf("%d commits", len(m.commits))
		return m, m.reloadDetail()
	case MainFiles:
		m.main = MainCommits
		m.files = nil
		m.filesCommitHash = ""
		m.openFilesPending = false
		m.focus = FocusMain
		m.status = fmt.Sprintf("%d commits", len(m.commits))
		return m, m.reloadDetail()
	default:
		return m, nil
	}
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.chordG {
		m.chordG = false
		switch key {
		case "r":
			return m.openRelations()
		case "g", "home":
			return m.gotoMainTop()
		case "f":
			if m.main == MainBlame {
				return m.followBlameLine()
			}
		}
		// Unrecognized second key — ignore chord, continue with key.
	}
	if key == "g" {
		m.chordG = true
		m.status = "g · g top · r relations"
		if m.main == MainBlame {
			m.status = "g · g top · r relations · f follow"
		}
		return m, nil
	}

	switch m.main {
	case MainCommits:
		return m.handleCommitKeys(msg)
	case MainFiles:
		return m.handleFileKeys(msg)
	case MainHistory:
		return m.handleHistoryKeys(msg)
	case MainBlame:
		return m.handleBlameKeys(msg)
	}
	return m, nil
}

func (m Model) gotoMainTop() (tea.Model, tea.Cmd) {
	switch m.main {
	case MainCommits:
		if len(m.commitIndices()) == 0 {
			return m, nil
		}
		m.cursor = 0
		m.ensureCommitVisible()
	case MainFiles:
		if len(m.fileIndices()) == 0 {
			return m, nil
		}
		m.fileCursor = 0
		m.ensureFileVisible()
	case MainHistory:
		if len(m.historyIndices()) == 0 {
			return m, nil
		}
		m.historyCursor = 0
		m.ensureHistoryVisible()
	case MainBlame:
		if len(m.blameIndices()) == 0 {
			return m, nil
		}
		m.blameCursor = 0
		m.ensureBlameVisible()
	}
	return m, m.reloadDetail()
}

func (m Model) handleCommitKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	idx := m.commitIndices()
	n := len(idx)
	switch msg.String() {
	case "h", "left":
		if m.commitCol > 0 {
			m.commitCol--
		}
		return m, nil
	case "l", "right":
		if m.commitCol < commitColCount-1 {
			m.commitCol++
		}
		return m, nil
	case "0":
		m.commitCol = 0
		return m, nil
	case "$":
		m.commitCol = commitColCount - 1
		return m, nil
	case "o":
		if m.commitCol == commitColGraph {
			return m, nil
		}
		if m.commitSortCol != m.commitCol {
			m.commitSortCol = m.commitCol
			m.commitSortDir = SortAsc
		} else {
			m.commitSortDir = CycleSort(m.commitSortDir)
			if m.commitSortDir == SortNone {
				m.commitSortCol = -1
			}
		}
		m.clampMainCursor()
		m.refreshStatus()
		return m, m.reloadDetail()
	}
	if n == 0 {
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.cursor < n-1 {
			m.cursor++
			m.ensureCommitVisible()
			return m, m.reloadDetail()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCommitVisible()
			return m, m.reloadDetail()
		}
	case "home":
		m.cursor = 0
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.cursor = n - 1
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.cursor = min(n-1, m.cursor+m.mainPage())
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.cursor = max(0, m.cursor-m.mainPage())
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "enter":
		hash := m.selectedHash()
		// Only seed the files grid from a whole-commit Show. Path-scoped
		// detail (after browsing files/history) still carries a single
		// FileChange and would open a one-row files view.
		if m.detail != nil && hashMatch(m.detail.Commit.Hash, hash) && m.detailPath == "" {
			m.enterFilesView()
			return m, m.reloadDetail()
		}
		m.openFilesPending = true
		m.status = "opening files…"
		m.detailFilterPath = ""
		m.loadingDetail = true
		m.detailOffset = 0
		return m, m.reloadDetailNow()
	}
	return m, nil
}

func (m *Model) enterFilesView() {
	if m.detail == nil {
		return
	}
	m.enterFilesViewFrom(*m.detail)
}

func (m *Model) enterFilesViewFrom(d git.CommitDetail) {
	m.filter = ""
	m.filterTyping = false
	m.main = MainFiles
	m.files = d.Files
	m.filesCommitHash = d.Commit.Hash
	m.fileCursor = 0
	m.fileOffset = 0
	m.fileCol = 0
	m.fileSortCol = -1
	m.fileSortDir = SortNone
	m.focus = FocusMain
	m.status = fmt.Sprintf("%d files in %s · enter history · b blame · esc back", len(m.files), d.Commit.ShortHash)
}

func (m Model) handleFileKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	idx := m.fileIndices()
	n := len(idx)
	switch msg.String() {
	case "h", "left":
		if m.fileCol > 0 {
			m.fileCol--
		}
		return m, nil
	case "l", "right":
		if m.fileCol < fileColCount-1 {
			m.fileCol++
		}
		return m, nil
	case "0":
		m.fileCol = 0
		return m, nil
	case "$":
		m.fileCol = fileColCount - 1
		return m, nil
	case "o":
		if m.fileSortCol != m.fileCol {
			m.fileSortCol = m.fileCol
			m.fileSortDir = SortAsc
		} else {
			m.fileSortDir = CycleSort(m.fileSortDir)
			if m.fileSortDir == SortNone {
				m.fileSortCol = -1
			}
		}
		m.clampMainCursor()
		m.refreshStatus()
		return m, m.reloadDetail()
	}
	if n == 0 {
		if msg.String() == "enter" || msg.String() == "b" {
			m.status = "no files match"
		}
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.fileCursor < n-1 {
			m.fileCursor++
			m.ensureFileVisible()
			return m, m.reloadDetail()
		}
	case "k", "up":
		if m.fileCursor > 0 {
			m.fileCursor--
			m.ensureFileVisible()
			return m, m.reloadDetail()
		}
	case "home":
		m.fileCursor = 0
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.fileCursor = n - 1
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.fileCursor = min(n-1, m.fileCursor+m.mainPage())
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.fileCursor = max(0, m.fileCursor-m.mainPage())
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "enter":
		path := m.files[idx[m.fileCursor]].Path
		m.historyPath = path
		m.historyCol = commitColHash
		m.historySortCol = -1
		m.historySortDir = SortNone
		m.loadingHistory = true
		m.status = fmt.Sprintf("loading history · %s", path)
		return m, loadHistoryCmd(m.repo, path)
	case "b":
		return m, m.startBlame(m.files[idx[m.fileCursor]].Path, m.filesCommitHash, MainFiles)
	}
	return m, nil
}

func (m Model) handleHistoryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	idx := m.historyIndices()
	n := len(idx)
	switch msg.String() {
	case "h", "left":
		if m.historyCol > 0 {
			m.historyCol--
		}
		return m, nil
	case "l", "right":
		if m.historyCol < commitColCount-1 {
			m.historyCol++
		}
		return m, nil
	case "0":
		m.historyCol = 0
		return m, nil
	case "$":
		m.historyCol = commitColCount - 1
		return m, nil
	case "o":
		if m.historyCol == commitColGraph {
			return m, nil
		}
		if m.historySortCol != m.historyCol {
			m.historySortCol = m.historyCol
			m.historySortDir = SortAsc
		} else {
			m.historySortDir = CycleSort(m.historySortDir)
			if m.historySortDir == SortNone {
				m.historySortCol = -1
			}
		}
		m.clampMainCursor()
		m.refreshStatus()
		return m, m.reloadDetail()
	}
	if n == 0 {
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.historyCursor < n-1 {
			m.historyCursor++
			m.ensureHistoryVisible()
			return m, m.reloadDetail()
		}
	case "k", "up":
		if m.historyCursor > 0 {
			m.historyCursor--
			m.ensureHistoryVisible()
			return m, m.reloadDetail()
		}
	case "home":
		m.historyCursor = 0
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.historyCursor = n - 1
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.historyCursor = min(n-1, m.historyCursor+m.mainPage())
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.historyCursor = max(0, m.historyCursor-m.mainPage())
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "b", "enter":
		return m, m.startBlame(m.historyPath, m.history[idx[m.historyCursor]].Hash, MainHistory)
	}
	return m, nil
}

func (m *Model) startBlame(path, rev string, from MainView) tea.Cmd {
	if path == "" || rev == "" {
		m.status = "cannot blame: missing path or revision"
		return nil
	}
	m.filter = ""
	m.filterTyping = false
	m.blamePath = path
	m.blameRev = rev
	m.blameFrom = from
	m.blamePreferLine = 0
	m.loadingBlame = true
	m.chordG = false
	m.status = fmt.Sprintf("loading blame · %s @ %s", path, shortHash(rev))
	return loadBlameCmd(m.repo, path, rev)
}

func (m Model) handleBlameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	idx := m.blameIndices()
	n := len(idx)
	switch key {
	case "h", "left":
		if m.blameCol > 0 {
			m.blameCol--
		}
		return m, nil
	case "l", "right":
		if m.blameCol < blameColCount-1 {
			m.blameCol++
		}
		return m, nil
	case "0":
		m.blameCol = 0
		return m, nil
	case "$":
		m.blameCol = blameColCount - 1
		return m, nil
	case "o":
		if m.blameSortCol != m.blameCol {
			m.blameSortCol = m.blameCol
			m.blameSortDir = SortAsc
		} else {
			m.blameSortDir = CycleSort(m.blameSortDir)
			if m.blameSortDir == SortNone {
				m.blameSortCol = -1
			}
		}
		m.clampMainCursor()
		m.refreshStatus()
		return m, m.reloadDetail()
	}
	if n == 0 {
		return m, nil
	}
	switch key {
	case "j", "down":
		if m.blameCursor < n-1 {
			m.blameCursor++
			m.ensureBlameVisible()
			return m, m.reloadDetail()
		}
	case "k", "up":
		if m.blameCursor > 0 {
			m.blameCursor--
			m.ensureBlameVisible()
			return m, m.reloadDetail()
		}
	case "home":
		m.blameCursor = 0
		m.ensureBlameVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.blameCursor = n - 1
		m.ensureBlameVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.blameCursor = min(n-1, m.blameCursor+m.mainPage())
		m.ensureBlameVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.blameCursor = max(0, m.blameCursor-m.mainPage())
		m.ensureBlameVisible()
		return m, m.reloadDetail()
	case "f":
		return m.followBlameLine()
	}
	return m, nil
}

func (m Model) followBlameLine() (tea.Model, tea.Cmd) {
	idx := m.blameIndices()
	if m.blameCursor < 0 || m.blameCursor >= len(idx) {
		return m, nil
	}
	line := m.blame[idx[m.blameCursor]]
	if line.PreviousHash == "" {
		m.status = "no earlier revision for this line"
		return m, nil
	}
	path := m.blamePath
	if line.PreviousPath != "" {
		path = line.PreviousPath
	}
	m.blamePath = path
	m.blameRev = line.PreviousHash
	m.blamePreferLine = line.Line
	m.loadingBlame = true
	m.status = fmt.Sprintf("follow · %s @ %s", path, shortHash(line.PreviousHash))
	return m, loadBlameCmd(m.repo, path, line.PreviousHash)
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	total := m.detailVisualCount()
	page := max(1, m.detailViewHeight()-1)
	switch msg.String() {
	case "j", "down":
		if m.detailOffset < total-1 {
			m.detailOffset++
		}
	case "k", "up":
		if m.detailOffset > 0 {
			m.detailOffset--
		}
	case "g", "home":
		m.detailOffset = 0
	case "G", "end":
		m.detailOffset = max(0, total-page)
	case "ctrl+d":
		m.detailOffset = min(max(0, total-page), m.detailOffset+page)
	case "ctrl+u":
		m.detailOffset = max(0, m.detailOffset-page)
	case "esc", "backspace", "h":
		return m.goBack()
	}
	return m, nil
}

// refresh reloads the commit log (and re-fetches the current view), matching
// ctrl+r. Shared by the keybinding and :refresh so the two cannot drift.
func (m *Model) refresh() tea.Cmd {
	if m.loading {
		return nil
	}
	m.chordG = false
	m.refreshPreferHash = m.commitListHash()
	m.loading = true
	m.err = ""
	m.status = "refreshing…"
	return loadCommitsCmd(m.repo)
}

func (m Model) commitListHash() string {
	idx := m.commitIndices()
	if m.cursor >= 0 && m.cursor < len(idx) {
		return m.commits[idx[m.cursor]].Hash
	}
	if m.main == MainFiles {
		return m.filesCommitHash
	}
	return ""
}

func (m *Model) restoreCommitCursor(hash string) {
	if hash == "" {
		m.cursor = 0
		m.commitOffset = 0
		return
	}
	idx := m.commitIndices()
	for i, src := range idx {
		if hashMatch(m.commits[src].Hash, hash) {
			m.cursor = i
			m.ensureCommitVisible()
			return
		}
	}
	m.cursor = 0
	m.commitOffset = 0
}

// afterRefreshCmds reloads the active view after a commit-log refresh.
func (m *Model) afterRefreshCmds() tea.Cmd {
	var cmds []tea.Cmd
	switch m.main {
	case MainHistory:
		if m.historyPath != "" {
			m.loadingHistory = true
			cmds = append(cmds, loadHistoryCmd(m.repo, m.historyPath))
		}
	case MainBlame:
		if m.blamePath != "" {
			idx := m.blameIndices()
			if m.blameCursor >= 0 && m.blameCursor < len(idx) {
				m.blamePreferLine = m.blame[idx[m.blameCursor]].Line
			}
			m.loadingBlame = true
			cmds = append(cmds, loadBlameCmd(m.repo, m.blamePath, m.blameRev))
		} else if c := m.reloadDetail(); c != nil {
			cmds = append(cmds, c)
		}
	default:
		if c := m.reloadDetail(); c != nil {
			cmds = append(cmds, c)
		}
	}
	if m.explorer.Opened() {
		if c := m.reloadRelationsCmd(); c != nil {
			cmds = append(cmds, c)
		}
	}
	switch len(cmds) {
	case 0:
		return nil
	case 1:
		return cmds[0]
	default:
		return tea.Batch(cmds...)
	}
}

func (m *Model) reloadRelationsCmd() tea.Cmd {
	m.loadingRel = true
	if m.main == MainBlame {
		idx := m.blameIndices()
		if m.blameCursor < 0 || m.blameCursor >= len(idx) {
			m.loadingRel = false
			return nil
		}
		line := m.blame[idx[m.blameCursor]]
		return loadLineRelationsCmd(m.repo, m.blamePath, m.blameRev, line)
	}
	hash := m.selectedHash()
	if hash == "" {
		m.loadingRel = false
		return nil
	}
	return loadCommitRelationsCmd(m.repo, hash)
}

// syncFilesFromDetail refreshes the files grid from a newly loaded detail
// payload, keeping the cursor on the same path when possible.
func (m *Model) syncFilesFromDetail() {
	if m.detail == nil {
		return
	}
	m.syncFilesFrom(m.detail.Files, m.detail.Commit.Hash)
}

func (m *Model) syncFilesFrom(files []git.FileChange, hash string) {
	prefer := ""
	idx := m.fileIndices()
	if m.fileCursor >= 0 && m.fileCursor < len(idx) {
		prefer = m.files[idx[m.fileCursor]].Path
	}
	m.files = files
	m.filesCommitHash = hash
	m.fileCursor = 0
	if prefer != "" {
		for i, src := range m.fileIndices() {
			if m.files[src].Path == prefer {
				m.fileCursor = i
				break
			}
		}
	}
	m.ensureFileVisible()
}

func (m *Model) ensureCommitVisible() {
	ensureVisible(&m.commitOffset, m.cursor, m.mainListHeight())
}

func (m *Model) ensureFileVisible() {
	ensureVisible(&m.fileOffset, m.fileCursor, m.mainListHeight())
}

func (m *Model) ensureHistoryVisible() {
	ensureVisible(&m.historyOffset, m.historyCursor, m.mainListHeight())
}

func (m *Model) ensureBlameVisible() {
	ensureVisible(&m.blameOffset, m.blameCursor, m.mainListHeight())
}

func ensureVisible(offset *int, cursor, height int) {
	if height <= 0 {
		return
	}
	if cursor < *offset {
		*offset = cursor
	}
	if cursor >= *offset+height {
		*offset = cursor - height + 1
	}
}

func (m Model) mainPage() int {
	return max(1, m.mainListHeight()-1)
}

func (m Model) mainListHeight() int {
	// All main grids use header + rule chrome.
	return max(1, m.paneContentHeight()-2)
}

func (m Model) detailViewHeight() int {
	return max(1, m.paneContentHeight())
}

func (m Model) bodyHeight() int {
	return max(1, m.height-1)
}

// paneContentHeight is the inner height available inside a bordered pane.
func (m Model) paneContentHeight() int {
	return max(1, m.bodyHeight()-borderOverhead)
}

func (m Model) mainPaneWidth() int {
	if m.width < 80 {
		return max(20, m.width)
	}
	return max(40, m.width/2)
}

func (m Model) detailInnerWidth() int {
	if m.width < 80 {
		return max(1, m.width-borderOverhead)
	}
	dw := m.width - m.mainPaneWidth()
	return max(1, dw-borderOverhead)
}

// detailVisualLines expands logical detail rows for the current wrap mode.
func (m Model) detailVisualLines() []string {
	width := m.detailInnerWidth()
	body := m.ensureDetailLogical(width)
	if m.detailWrap {
		return m.ensureDetailVisual(width, body)
	}
	return expandDetailRows(body, width, false)
}

func (m Model) detailVisualCount() int {
	width := m.detailInnerWidth()
	body := m.ensureDetailLogical(width)
	if m.detailWrap {
		return len(m.ensureDetailVisual(width, body))
	}
	return len(body)
}

func (m Model) borderForFocus(f Focus) lipgloss.Color {
	if m.focus == f {
		return colorBorderFocused
	}
	return colorBorderUnfocused
}

// framePane wraps pane content in a square border; focused panes use a
// stronger slate frame so selection chrome stays neutral.
func (m Model) framePane(content string, outerW, outerH int, focus Focus) string {
	innerW := max(1, outerW-borderOverhead)
	innerH := max(1, outerH-borderOverhead)
	return lipgloss.NewStyle().
		Width(innerW).
		Height(innerH).
		Border(panelBorder()).
		BorderForeground(m.borderForFocus(focus)).
		Render(content)
}

func shortHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

func hashMatch(full, expect string) bool {
	if expect == "" {
		return false
	}
	f := strings.ToLower(full)
	e := strings.ToLower(expect)
	return f == e || strings.HasPrefix(f, e) || strings.HasPrefix(e, f)
}

func identityIndices(n int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return idx
}

func (m Model) commitIndices() []int {
	col := m.commitCol
	if m.filter != "" || m.filterTyping {
		col = m.commitFilterCol
	}
	idx := filterCommitIndicesCol(m.commits, m.filter, col)
	return sortCommitIndices(m.commits, idx, m.commitSortCol, m.commitSortDir)
}

func (m Model) fileIndices() []int {
	col := -1
	if m.filter != "" || m.filterTyping {
		col = m.fileFilterCol
	}
	idx := filterFileIndicesCol(m.files, m.filter, col)
	return sortFileIndices(m.files, idx, m.fileSortCol, m.fileSortDir)
}

func (m Model) historyIndices() []int {
	col := -1
	if m.filter != "" || m.filterTyping {
		col = m.historyFilterCol
	}
	idx := filterCommitIndicesCol(m.history, m.filter, col)
	return sortCommitIndices(m.history, idx, m.historySortCol, m.historySortDir)
}

func (m Model) blameIndices() []int {
	col := -1
	if m.filter != "" || m.filterTyping {
		col = m.blameFilterCol
	}
	idx := filterBlameIndicesCol(m.blame, m.filter, col)
	return sortBlameIndices(m.blame, idx, m.blameSortCol, m.blameSortDir)
}

func (m *Model) clampMainCursor() {
	switch m.main {
	case MainCommits:
		m.cursor = min(m.cursor, max(0, len(m.commitIndices())-1))
		m.ensureCommitVisible()
	case MainFiles:
		m.fileCursor = min(m.fileCursor, max(0, len(m.fileIndices())-1))
		m.ensureFileVisible()
	case MainHistory:
		m.historyCursor = min(m.historyCursor, max(0, len(m.historyIndices())-1))
		m.ensureHistoryVisible()
	case MainBlame:
		m.blameCursor = min(m.blameCursor, max(0, len(m.blameIndices())-1))
		m.ensureBlameVisible()
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "…"
	}
	if m.help.Visible() {
		return m.help.View()
	}
	var b strings.Builder
	b.WriteString(m.renderBody())
	b.WriteByte('\n')
	b.WriteString(m.renderStatus())
	view := clampFrame(b.String(), m.height, m.width)

	// Centered creel-style command palette popup over the workspace.
	if m.palette.IsVisible() {
		pw, ph := palettePopupDim()
		palPanel := m.palette.View(pw, ph)
		panelW := lipgloss.Width(palPanel)
		panelH := lipgloss.Height(palPanel)
		panelX := (m.width - panelW) / 2
		panelY := (m.height - 1 - panelH) / 2
		if panelY < 0 {
			panelY = 0
		}
		view = placeOverlay(view, palPanel, panelX, panelY)
	}
	return view
}

func (m Model) renderStatus() string {
	if m.exTyping {
		return fitWidth(styleFilter.Render(" :"+m.exLine+"█")+"  "+styleMuted.Render("enter run · esc cancel"), m.width)
	}
	if m.filterTyping {
		return fitWidth(styleFilter.Render(" "+m.filterPrompt())+"  "+styleMuted.Render("enter keep · esc clear"), m.width)
	}

	leftTab := renderStatusTab(m.mainTitle(), m.focus == FocusMain)
	rightTitle := "detail"
	rightFocused := m.focus == FocusDetail
	if m.explorer.Opened() {
		rightTitle = "relationships"
		rightFocused = m.focus == FocusExplorer
	}
	rightTab := renderStatusTab(rightTitle, rightFocused)

	repo := filepathBase(m.repo.Path)
	midParts := []string{styleTitle.Render(repo)}
	if m.branch != "" || m.head != "" {
		midParts = append(midParts, styleMuted.Render(fmt.Sprintf("%s @ %s", m.branch, m.head)))
	}

	busy := m.loading || m.loadingHistory || m.loadingBlame || m.loadingRel || m.loadingDetail

	var hints string
	switch {
	case m.err != "":
		midParts = append(midParts, styleErr.Render(m.err))
	case busy:
		midParts = append(midParts, styleMuted.Render(m.status+" · fetching…"))
	default:
		if m.status != "" {
			midParts = append(midParts, styleMuted.Render(m.status))
		}
		hints = styleMuted.Render(statusHints(m.main, m.explorer.Opened()))
	}

	mid := strings.Join(midParts, styleMuted.Render(" · "))
	left := leftTab
	if mid != "" {
		left += " " + mid
	}

	// Narrow layout: only the visible pane is on screen — keep its tab on the left.
	if m.width < 80 {
		if m.explorer.Opened() {
			left = renderStatusTab(rightTitle, rightFocused)
			if mid != "" {
				left += " " + mid
			}
		}
		return fitWidth(left, m.width)
	}

	right := rightTab
	if hints != "" {
		right = hints + " " + rightTab
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		avail := m.width - lipgloss.Width(left) - lipgloss.Width(rightTab) - 1
		if avail < 8 {
			gap = 1
			right = rightTab
			if lipgloss.Width(left)+1+lipgloss.Width(right) > m.width {
				return fitWidth(left, m.width)
			}
		} else {
			hints = styleMuted.Render(fitWidth(statusHints(m.main, m.explorer.Opened()), avail))
			right = hints + " " + rightTab
			gap = m.width - lipgloss.Width(left) - lipgloss.Width(right)
			if gap < 1 {
				gap = 1
			}
		}
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderBody() string {
	h := m.bodyHeight()
	if m.loading && len(m.commits) == 0 {
		return fitWidth(styleMuted.Render(" loading commit history…"), m.width)
	}
	innerH := m.paneContentHeight()
	if m.width < 80 {
		innerW := max(1, m.width-borderOverhead)
		if m.explorer.Opened() {
			m.explorer.SetSize(innerW, innerH)
			return m.framePane(m.explorer.View(m.focus == FocusExplorer), m.width, h, FocusExplorer)
		}
		return m.framePane(m.renderMainPane(innerW, innerH), m.width, h, FocusMain)
	}
	cw := m.mainPaneWidth()
	dw := m.width - cw
	leftInner := max(1, cw-borderOverhead)
	rightInner := max(1, dw-borderOverhead)
	left := m.framePane(m.renderMainPane(leftInner, innerH), cw, h, FocusMain)
	var right string
	if m.explorer.Opened() {
		m.explorer.SetSize(rightInner, innerH)
		right = m.framePane(m.explorer.View(m.focus == FocusExplorer), dw, h, FocusExplorer)
	} else {
		right = m.framePane(m.renderDetailPane(rightInner, innerH), dw, h, FocusDetail)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderMainPane(width, height int) string {
	switch m.main {
	case MainFiles:
		return m.renderFilesPane(width, height)
	case MainHistory:
		return m.renderHistoryPane(width, height)
	case MainBlame:
		return m.renderBlamePane(width, height)
	default:
		return m.renderCommitPane(width, height)
	}
}

func (m Model) mainTitle() string {
	switch m.main {
	case MainFiles:
		return "files"
	case MainHistory:
		return "history"
	case MainBlame:
		return "blame"
	default:
		return "commits"
	}
}

// renderStatusTab draws a creel-style pane label for the status bar: blue/white
// pill when focused, muted text when not.
func renderStatusTab(title string, focused bool) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	if focused {
		return styleSelected.Render(title)
	}
	return styleMuted.Padding(0, 1).Render(title)
}

func (m Model) renderCommitPane(width, height int) string {
	idx := m.commitIndices()
	graphs := commitGraphLines(m.commits, idx, m.commitSortDir)
	rows := make([][]string, len(idx))
	for i, src := range idx {
		rows[i] = commitRow(m.commits[src], graphs[i])
	}
	g := Grid{
		Columns:   commitColumns,
		Rows:      rows,
		CursorRow: m.cursor,
		CursorCol: m.commitCol,
		OffsetRow: m.commitOffset,
		SortCol:   m.commitSortCol,
		SortDir:   m.commitSortDir,
		Width:     width,
		Height:    height,
		Focused:   m.focus == FocusMain,
		CellStyle: styleCommitGraphCell,
	}
	g.AutoWidths()
	g.ClampCursor()
	return g.View()
}

func (m Model) renderFilesPane(width, height int) string {
	idx := m.fileIndices()
	rows := make([][]string, len(idx))
	for i, src := range idx {
		rows[i] = fileRow(m.files[src])
	}
	g := Grid{
		Columns:   fileColumns,
		Rows:      rows,
		CursorRow: m.fileCursor,
		CursorCol: m.fileCol,
		OffsetRow: m.fileOffset,
		SortCol:   m.fileSortCol,
		SortDir:   m.fileSortDir,
		Width:     width,
		Height:    height,
		Focused:   m.focus == FocusMain,
	}
	g.AutoWidths()
	g.ClampCursor()
	return g.View()
}

func (m Model) renderHistoryPane(width, height int) string {
	idx := m.historyIndices()
	graphs := commitGraphLines(m.history, idx, m.historySortDir)
	rows := make([][]string, len(idx))
	for i, src := range idx {
		rows[i] = commitRow(m.history[src], graphs[i])
	}
	g := Grid{
		Columns:   commitColumns,
		Rows:      rows,
		CursorRow: m.historyCursor,
		CursorCol: m.historyCol,
		OffsetRow: m.historyOffset,
		SortCol:   m.historySortCol,
		SortDir:   m.historySortDir,
		Width:     width,
		Height:    height,
		Focused:   m.focus == FocusMain,
		CellStyle: styleCommitGraphCell,
	}
	g.AutoWidths()
	g.ClampCursor()
	return g.View()
}

func (m Model) renderBlamePane(width, height int) string {
	idx := m.blameIndices()
	rows := make([][]string, len(idx))
	for i, src := range idx {
		rows[i] = blameRow(m.blame[src])
	}
	newest, oldest := blameAgeRange(m.blame)
	g := Grid{
		Columns:             blameColumns,
		Rows:                rows,
		CursorRow:           m.blameCursor,
		CursorCol:           m.blameCol,
		OffsetRow:           m.blameOffset,
		SortCol:             m.blameSortCol,
		SortDir:             m.blameSortDir,
		Width:               width,
		Height:              height,
		Focused:             m.focus == FocusMain,
		NoStripe:            true,
		SoftCursor:          true,
		SkipCursorPaintCols: []int{blameColCode},
		MuteCols:            []int{blameColLine, blameColCommit, blameColAge, blameColAuthor},
		CellStyle: func(row, col int, text string) (string, bool) {
			if col != blameColCode || row < 0 || row >= len(idx) {
				return "", false
			}
			bl := m.blame[idx[row]]
			return blameAgeStyle(bl.When, newest, oldest).Render(text), true
		},
	}
	g.AutoWidths()
	g.ClampCursor()
	return g.View()
}

func padPane(lines []string, width, height int) string {
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines[:height], "\n")
}

func relativeAge(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return "just now"
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

func blameAgeRange(lines []git.BlameLine) (newest, oldest time.Time) {
	for _, l := range lines {
		if l.When.IsZero() {
			continue
		}
		if newest.IsZero() || l.When.After(newest) {
			newest = l.When
		}
		if oldest.IsZero() || l.When.Before(oldest) {
			oldest = l.When
		}
	}
	return newest, oldest
}

// Soft age washes for light terminals — newer = cooler mint, older = warmer parchment.
func blameAgeStyle(when, newest, oldest time.Time) lipgloss.Style {
	if when.IsZero() || newest.IsZero() || oldest.IsZero() || !newest.After(oldest) {
		return lipgloss.NewStyle()
	}
	span := newest.Sub(oldest).Seconds()
	if span <= 0 {
		return lipgloss.NewStyle()
	}
	// 0 = newest, 1 = oldest
	t := newest.Sub(when).Seconds() / span
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	switch {
	case t < 0.33:
		return lipgloss.NewStyle().Background(lipgloss.Color("#eef6f0")).Foreground(lipgloss.Color("#24292f"))
	case t < 0.66:
		return lipgloss.NewStyle().Background(lipgloss.Color("#f6f1e7")).Foreground(lipgloss.Color("#24292f"))
	default:
		return lipgloss.NewStyle().Background(lipgloss.Color("#f0e6e4")).Foreground(lipgloss.Color("#24292f"))
	}
}

func (m Model) renderDetailPane(width, height int) string {
	h := height
	if h < 1 {
		h = 1
	}
	lines, _ := m.detailVisualWindow(width, h, m.detailOffset)
	return padPane(lines, width, height)
}

func (m Model) detailLines() []string {
	if m.detail == nil {
		// Stay blank while the first detail load is in flight — avoids flicker.
		if m.loadingDetail {
			return nil
		}
		return []string{styleMuted.Render("(no commit selected)")}
	}
	d := m.detail
	if m.diffMode == DiffZen {
		return m.detailLinesZen(d)
	}
	return m.detailLinesUnified(d)
}

// detailLinesZen keeps chrome to a bare minimum: subject, then the patch
// (file banners + code). Hash/author/stats stay out of the way for scanning.
func (m Model) detailLinesZen(d *git.CommitDetail) []string {
	var out []string
	pad := strings.Repeat(" ", cellPad)
	if m.main == MainBlame {
		idx := m.blameIndices()
		if m.blameCursor >= 0 && m.blameCursor < len(idx) {
			bl := m.blame[idx[m.blameCursor]]
			out = append(out, styleMuted.Render(pad+fmt.Sprintf("line %d · %s", bl.Line, relativeAge(bl.When))))
		}
	}
	out = append(out, pad+styleTitle.Render(d.Commit.Subject))
	if diff := strings.TrimRight(d.Diff, "\n"); diff != "" {
		out = append(out, zenDiffLines(diff, m.zenContext)...)
	} else if !m.detailPatchPending {
		out = append(out, styleMuted.Render(pad+"(no patch)"))
	}
	return out
}

func (m Model) detailLinesUnified(d *git.CommitDetail) []string {
	var out []string
	pad := func(s string) string { return strings.Repeat(" ", cellPad) + s }

	// Optional blame context sits above commit meta.
	if m.main == MainBlame {
		idx := m.blameIndices()
		if m.blameCursor >= 0 && m.blameCursor < len(idx) {
			bl := m.blame[idx[m.blameCursor]]
			out = append(out, styleMuted.Render(pad(fmt.Sprintf("line %d · %s", bl.Line, relativeAge(bl.When)))))
			if bl.Summary != "" {
				out = append(out, pad(styleTitle.Render(bl.Summary)))
			}
			out = append(out, pad(""))
		}
	}

	// Meta block — quiet; hash keeps semantic blue.
	out = append(out, pad(styleHash.Render(d.Commit.Hash)))
	out = append(out, styleMuted.Render(pad(fmt.Sprintf("%s <%s>", d.Commit.Author, d.Commit.Email))))
	out = append(out, styleMuted.Render(pad(d.Commit.Date.Local().Format(time.RFC1123))))
	if m.detailFilterPath != "" {
		out = append(out, styleMuted.Render(pad("path  "+m.detailFilterPath)))
	}

	// Subject is the hero line.
	out = append(out, pad(""))
	out = append(out, pad(styleTitle.Render(d.Commit.Subject)))
	if d.Body != "" {
		out = append(out, pad(""))
		for _, line := range strings.Split(d.Body, "\n") {
			out = append(out, styleMuted.Render(pad(line)))
		}
	}

	// Full-bleed rule under the message; content above/below stays inset.
	out = append(out, detailSepMarker)
	out = append(out, pad(fmt.Sprintf("%s  %s  %s  %s",
		styleMuted.Render(fmt.Sprintf("%d files", d.Commit.Files)),
		styleAdd.Render(fmt.Sprintf("+%d", d.Commit.Additions)),
		styleDel.Render(fmt.Sprintf("-%d", d.Commit.Deletions)),
		styleMuted.Render(fmt.Sprintf("· %s ·U%d · D [/] · w", m.diffMode.Label(), m.zenContext)),
	)))
	if stat := strings.TrimRight(d.Stat, "\n"); stat != "" {
		for _, line := range strings.Split(stat, "\n") {
			out = append(out, styleMuted.Render(pad(line)))
		}
	}
	if diff := strings.TrimRight(d.Diff, "\n"); diff != "" {
		out = append(out, pad(""))
		for _, line := range strings.Split(diff, "\n") {
			out = append(out, pad(line))
		}
	}
	return out
}

// detailSepMarker is expanded to a full-width muted rule in renderDetailLine.
const detailSepMarker = "\x1edetail-sep"

func expandDetailRows(body []string, width int, wrap bool) []string {
	out := make([]string, 0, len(body))
	for _, line := range body {
		out = append(out, renderDetailRows(line, width, wrap)...)
	}
	return out
}

func renderDetailLine(line string, width int) string {
	rows := renderDetailRows(line, width, false)
	if len(rows) == 0 {
		return fitWidth("", width)
	}
	return rows[0]
}

func renderDetailRows(line string, width int, wrap bool) []string {
	// Git diffs from CRLF files keep a trailing \r after Split(..., "\n").
	// A carriage return mid-row sends the cursor to column 0, so wash padding
	// then paints over the left pane.
	line = strings.ReplaceAll(line, "\r", "")
	if line == detailSepMarker || line == zenFileRuleMarker {
		return []string{fitWidth(styleGridBorder.Render(strings.Repeat("─", max(0, width))), width)}
	}
	switch {
	case strings.HasPrefix(line, zenFilePrefix):
		return renderZenFile(strings.TrimPrefix(line, zenFilePrefix), width, wrap)
	case strings.HasPrefix(line, zenHunkPrefix):
		return renderZenHunk(strings.TrimPrefix(line, zenHunkPrefix), width, wrap)
	case strings.HasPrefix(line, zenCtxPrefix):
		if num, text, ok := parseZenNumText(strings.TrimPrefix(line, zenCtxPrefix)); ok {
			return renderZenContext(num, text, width, wrap)
		}
	case strings.HasPrefix(line, zenAddPrefix):
		if num, text, ok := parseZenNumText(strings.TrimPrefix(line, zenAddPrefix)); ok {
			return renderZenChange(true, num, text, width, wrap)
		}
	case strings.HasPrefix(line, zenDelPrefix):
		if num, text, ok := parseZenNumText(strings.TrimPrefix(line, zenDelPrefix)); ok {
			return renderZenChange(false, num, text, width, wrap)
		}
	}
	if strings.Contains(line, "\x1b[") {
		return []string{fitDetailInset(line, width)}
	}
	// Unified / plain rows may carry a leading cellPad from detailLinesUnified.
	content := line
	if p := strings.Repeat(" ", cellPad); strings.HasPrefix(line, p) {
		content = line[len(p):]
	}
	switch {
	case isDiffFileHeader(content):
		return renderWrappedCell(styleDiffMeta, content, width, wrap)
	case strings.HasPrefix(content, "@@"):
		return renderWrappedCell(styleDiffHunk, content, width, wrap)
	case strings.HasPrefix(content, "+") && !strings.HasPrefix(content, "+++"):
		return renderWrappedCell(styleAddWash, content, width, wrap)
	case strings.HasPrefix(content, "-") && !strings.HasPrefix(content, "---"):
		return renderWrappedCell(styleDelWash, content, width, wrap)
	default:
		return renderWrappedCell(lipgloss.NewStyle(), content, width, wrap)
	}
}

// fitDetailInset keeps one cell of empty space on the right; left pad is
// expected to already be present in s (from detailLinesUnified / zen subject).
func fitDetailInset(s string, width int) string {
	inner := width - cellPad
	if inner < 1 {
		return fitWidth(s, width)
	}
	return fitWidth(fitWidth(s, inner), width)
}

func renderWrappedCell(st lipgloss.Style, text string, width int, wrap bool) []string {
	pad := strings.Repeat(" ", cellPad)
	bodyW := width - 2*cellPad
	if bodyW < 1 {
		bodyW = max(1, width-cellPad)
	}
	if !wrap {
		return []string{fitWidth(pad+cell(st, text, bodyW), width)}
	}
	chunks := wrapDisplay(text, bodyW)
	rows := make([]string, 0, len(chunks))
	for _, c := range chunks {
		rows = append(rows, fitWidth(pad+cell(st, c, bodyW), width))
	}
	return rows
}

func isDiffFileHeader(line string) bool {
	switch {
	case strings.HasPrefix(line, "diff --git"),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "--- "),
		strings.HasPrefix(line, "+++ "),
		strings.HasPrefix(line, "new file mode"),
		strings.HasPrefix(line, "deleted file mode"),
		strings.HasPrefix(line, "old mode"),
		strings.HasPrefix(line, "new mode"),
		strings.HasPrefix(line, "similarity index"),
		strings.HasPrefix(line, "rename from"),
		strings.HasPrefix(line, "rename to"),
		strings.HasPrefix(line, "copy from"),
		strings.HasPrefix(line, "copy to"):
		return true
	default:
		return false
	}
}

// cell renders a single-line pane cell. Inline(true) is required: lipgloss
// Width() wraps before MaxWidth truncates, which turns a long commit subject
// into multiple rows, pushes the frame past the terminal height, and scrolls
// the title away.
func cell(base lipgloss.Style, s string, width int) string {
	if width <= 0 {
		return ""
	}
	return base.Inline(true).Width(width).MaxWidth(width).Render(s)
}

func fitWidth(s string, width int) string {
	return cell(lipgloss.NewStyle(), s, width)
}

func clampFrame(s string, height, width int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lipgloss.Width(lines[i]) != width {
			lines[i] = fitWidth(lines[i], width)
		}
	}
	switch {
	case len(lines) < height:
		pad := strings.Repeat(" ", max(0, width))
		for len(lines) < height {
			lines = append(lines, pad)
		}
	case len(lines) > height:
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func filepathBase(p string) string {
	p = strings.TrimRight(p, "/\\")
	if i := strings.LastIndexAny(p, "/\\"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// Run starts the Bubble Tea program.
func Run(repo *git.Repo) error {
	p := tea.NewProgram(New(repo), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
