// Package ui is the Bubble Tea front-end for ore.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rsiota/ore/internal/git"
)

// Focus is which pane receives keys.
type Focus int

const (
	FocusMain Focus = iota
	FocusDetail
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

	commits      []git.Commit
	cursor       int
	commitOffset int

	files            []git.FileChange
	fileCursor       int
	fileOffset       int
	filesCommitHash  string
	openFilesPending bool

	historyPath    string
	history        []git.Commit
	historyCursor  int
	historyOffset  int
	loadingHistory bool

	blame          []git.BlameLine
	blamePath      string
	blameRev       string
	blameCursor    int
	blameOffset    int
	blameFrom      MainView // MainFiles or MainHistory
	blamePreferLine int
	loadingBlame   bool
	chordG         bool // pending g-prefix for gg / gf in blame

	detail           *git.CommitDetail
	detailFilterPath string // path passed to Show/ShowPath for stale checks
	detailOffset     int
	loading          bool
	loadingDetail    bool
	err              string
	status           string

	branch string
	head   string
}

// New builds a model bound to repo. Call Init via the Bubble Tea program.
func New(repo *git.Repo) Model {
	return Model{
		repo:    repo,
		focus:   FocusMain,
		main:    MainCommits,
		loading: true,
		status:  "loading commits…",
	}
}

type commitsLoadedMsg struct {
	commits []git.Commit
	branch  string
	head    string
	err     error
}

type detailLoadedMsg struct {
	hash   string
	path   string
	detail git.CommitDetail
	err    error
}

type historyLoadedMsg struct {
	path    string
	commits []git.Commit
	err     error
}

type blameLoadedMsg struct {
	path    string
	rev     string
	lines   []git.BlameLine
	err     error
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

func loadDetailCmd(repo *git.Repo, hash, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var (
			detail git.CommitDetail
			err    error
		)
		if path != "" {
			detail, err = repo.ShowPath(ctx, hash, path)
		} else {
			detail, err = repo.Show(ctx, hash)
		}
		return detailLoadedMsg{hash: hash, path: path, detail: detail, err: err}
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
		return m, nil

	case commitsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "error"
			return m, nil
		}
		m.commits = msg.commits
		m.branch = msg.branch
		m.head = msg.head
		m.cursor = 0
		m.commitOffset = 0
		m.main = MainCommits
		m.status = fmt.Sprintf("%d commits", len(m.commits))
		if len(m.commits) > 0 {
			return m, m.reloadDetail()
		}
		return m, nil

	case detailLoadedMsg:
		m.loadingDetail = false
		if msg.hash != m.selectedHash() || msg.path != m.detailFilterPath {
			return m, nil // stale
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.detail = nil
			return m, nil
		}
		m.err = ""
		d := msg.detail
		m.detail = &d
		m.detailOffset = 0
		if m.openFilesPending && m.main == MainCommits {
			m.openFilesPending = false
			m.enterFilesView()
			return m, m.reloadDetail()
		}
		return m, nil

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
		m.main = MainHistory
		m.focus = FocusMain
		m.status = fmt.Sprintf("history · %s · %d commits · b/enter blame", m.historyPath, len(m.history))
		if len(m.history) > 0 {
			return m, m.reloadDetail()
		}
		m.detail = nil
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
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) selectedHash() string {
	switch m.main {
	case MainBlame:
		if len(m.blame) == 0 {
			return ""
		}
		return m.blame[m.blameCursor].Hash
	case MainHistory:
		if len(m.history) == 0 {
			return ""
		}
		return m.history[m.historyCursor].Hash
	case MainFiles:
		return m.filesCommitHash
	default:
		if len(m.commits) == 0 {
			return ""
		}
		return m.commits[m.cursor].Hash
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.chordG = false
		if m.focus == FocusMain {
			m.focus = FocusDetail
		} else {
			m.focus = FocusMain
		}
		return m, nil
	case "esc", "backspace":
		m.chordG = false
		return m.goBack()
	case "?":
		m.chordG = false
		m.status = "enter open · b blame · f follow line · esc back · j/k · tab · q"
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

func (m Model) handleCommitKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.commits) == 0 {
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.commits)-1 {
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
	case "g", "home":
		m.cursor = 0
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.cursor = len(m.commits) - 1
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.cursor = min(len(m.commits)-1, m.cursor+m.mainPage())
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.cursor = max(0, m.cursor-m.mainPage())
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "enter", "l":
		if m.detail != nil && m.detail.Commit.Hash == m.commits[m.cursor].Hash {
			m.enterFilesView()
			return m, m.reloadDetail()
		}
		m.openFilesPending = true
		m.status = "opening files…"
		return m, m.reloadDetail()
	}
	return m, nil
}

func (m *Model) enterFilesView() {
	if m.detail == nil {
		return
	}
	m.main = MainFiles
	m.files = m.detail.Files
	m.filesCommitHash = m.detail.Commit.Hash
	m.fileCursor = 0
	m.fileOffset = 0
	m.focus = FocusMain
	m.status = fmt.Sprintf("%d files in %s · enter history · b blame · esc back", len(m.files), m.detail.Commit.ShortHash)
}

func (m Model) handleFileKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.files) == 0 {
		if msg.String() == "enter" || msg.String() == "l" {
			m.status = "no files in this commit"
		}
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.fileCursor < len(m.files)-1 {
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
	case "g", "home":
		m.fileCursor = 0
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.fileCursor = len(m.files) - 1
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.fileCursor = min(len(m.files)-1, m.fileCursor+m.mainPage())
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.fileCursor = max(0, m.fileCursor-m.mainPage())
		m.ensureFileVisible()
		return m, m.reloadDetail()
	case "enter", "l":
		path := m.files[m.fileCursor].Path
		m.historyPath = path
		m.loadingHistory = true
		m.status = fmt.Sprintf("loading history · %s", path)
		return m, loadHistoryCmd(m.repo, path)
	case "b":
		return m.startBlame(m.files[m.fileCursor].Path, m.filesCommitHash, MainFiles)
	}
	return m, nil
}

func (m Model) handleHistoryKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.historyCursor < len(m.history)-1 {
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
	case "g", "home":
		m.historyCursor = 0
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.historyCursor = len(m.history) - 1
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.historyCursor = min(len(m.history)-1, m.historyCursor+m.mainPage())
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.historyCursor = max(0, m.historyCursor-m.mainPage())
		m.ensureHistoryVisible()
		return m, m.reloadDetail()
	case "b", "enter", "l":
		return m.startBlame(m.historyPath, m.history[m.historyCursor].Hash, MainHistory)
	}
	return m, nil
}

func (m Model) startBlame(path, rev string, from MainView) (tea.Model, tea.Cmd) {
	if path == "" || rev == "" {
		m.status = "cannot blame: missing path or revision"
		return m, nil
	}
	m.blamePath = path
	m.blameRev = rev
	m.blameFrom = from
	m.blamePreferLine = 0
	m.loadingBlame = true
	m.chordG = false
	m.status = fmt.Sprintf("loading blame · %s @ %s", path, shortHash(rev))
	return m, loadBlameCmd(m.repo, path, rev)
}

func (m Model) handleBlameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// g-prefix: gg = top, gf = follow line backward
	if m.chordG {
		m.chordG = false
		switch key {
		case "g", "home":
			if len(m.blame) == 0 {
				return m, nil
			}
			m.blameCursor = 0
			m.ensureBlameVisible()
			return m, m.reloadDetail()
		case "f":
			return m.followBlameLine()
		default:
			// fall through and handle key normally
		}
	}

	if len(m.blame) == 0 {
		return m, nil
	}
	switch key {
	case "j", "down":
		if m.blameCursor < len(m.blame)-1 {
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
	case "g":
		m.chordG = true
		m.status = "g · g top · f follow"
		return m, nil
	case "home":
		m.blameCursor = 0
		m.ensureBlameVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.blameCursor = len(m.blame) - 1
		m.ensureBlameVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.blameCursor = min(len(m.blame)-1, m.blameCursor+m.mainPage())
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
	if len(m.blame) == 0 {
		return m, nil
	}
	line := m.blame[m.blameCursor]
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
	lines := m.detailLines()
	page := max(1, m.detailViewHeight()-1)
	switch msg.String() {
	case "j", "down":
		if m.detailOffset < len(lines)-1 {
			m.detailOffset++
		}
	case "k", "up":
		if m.detailOffset > 0 {
			m.detailOffset--
		}
	case "g", "home":
		m.detailOffset = 0
	case "G", "end":
		m.detailOffset = max(0, len(lines)-page)
	case "ctrl+d":
		m.detailOffset = min(max(0, len(lines)-page), m.detailOffset+page)
	case "ctrl+u":
		m.detailOffset = max(0, m.detailOffset-page)
	case "esc", "backspace", "h":
		return m.goBack()
	}
	return m, nil
}

func (m *Model) reloadDetail() tea.Cmd {
	hash := m.selectedHash()
	if hash == "" {
		return nil
	}
	path := ""
	switch m.main {
	case MainFiles:
		if len(m.files) > 0 {
			path = m.files[m.fileCursor].Path
		}
	case MainHistory:
		path = m.historyPath
	case MainBlame:
		path = m.blamePath
	}
	m.detailFilterPath = path
	m.loadingDetail = true
	m.detailOffset = 0
	return loadDetailCmd(m.repo, hash, path)
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
	return max(1, m.bodyHeight()-2)
}

func (m Model) detailViewHeight() int {
	return max(1, m.bodyHeight())
}

func (m Model) bodyHeight() int {
	return max(1, m.height-3)
}

func (m Model) mainPaneWidth() int {
	if m.width < 80 {
		return max(20, m.width)
	}
	return max(40, m.width*55/100)
}

func shortHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "…"
	}
	var b strings.Builder
	b.WriteString(m.renderTitle())
	b.WriteByte('\n')
	b.WriteString(m.renderBody())
	b.WriteByte('\n')
	b.WriteString(m.renderStatus())
	return clampFrame(b.String(), m.height, m.width)
}

var (
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("180"))
	styleMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleFocus  = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("236"))
	styleHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true)
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleHash   = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))

	styleAdd = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a7f37"))
	styleDel = lipgloss.NewStyle().Foreground(lipgloss.Color("#cf222e"))

	styleAddWash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a7f37")).
			Background(lipgloss.Color("#dafbe1"))
	styleDelWash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cf222e")).
			Background(lipgloss.Color("#ffebe9"))
)

func (m Model) renderTitle() string {
	name := filepathBase(m.repo.Path)
	left := styleTitle.Render("ore") + "  " + styleMuted.Render(name)
	right := styleMuted.Render(fmt.Sprintf("%s @ %s", m.branch, m.head))
	pad := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		return fitWidth(left, m.width)
	}
	return left + strings.Repeat(" ", pad) + right
}

func (m Model) renderStatus() string {
	msg := m.status
	busy := m.loading || m.loadingDetail || m.loadingHistory || m.loadingBlame
	if m.err != "" {
		msg = styleErr.Render(m.err)
	} else if busy {
		msg = styleMuted.Render(msg + " · fetching…")
	} else {
		msg = styleMuted.Render(msg + "  ·  ? help  ·  q quit")
	}
	return fitWidth(msg, m.width)
}

func (m Model) renderBody() string {
	h := m.bodyHeight()
	if m.loading && len(m.commits) == 0 {
		return fitWidth(styleMuted.Render(" loading commit history…"), m.width)
	}
	if m.width < 80 {
		return m.renderMainPane(m.width, h)
	}
	cw := m.mainPaneWidth()
	dw := m.width - cw - 1
	left := m.renderMainPane(cw, h)
	right := m.renderDetailPane(dw, h)
	sep := styleMuted.Width(1).Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
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
		return " files"
	case MainHistory:
		return " history"
	case MainBlame:
		return " blame"
	default:
		return " commits"
	}
}

func (m Model) renderPaneHeader(width int, title string) []string {
	var lines []string
	if m.focus == FocusMain {
		lines = append(lines, cell(styleFocus, title, width))
	} else {
		lines = append(lines, cell(styleMuted, title, width))
	}
	return lines
}

func (m Model) renderCommitPane(width, height int) string {
	lines := m.renderPaneHeader(width, m.mainTitle())
	lines = append(lines, cell(styleHeader,
		fmt.Sprintf("%-7s %-10s %-12s %s", "HASH", "DATE", "AUTHOR", "SUBJECT"),
		width,
	))

	h := height - 2
	if h < 1 {
		h = 1
	}
	end := min(len(m.commits), m.commitOffset+h)
	for i := m.commitOffset; i < end; i++ {
		row := formatCommitRow(m.commits[i])
		if i == m.cursor {
			lines = append(lines, cell(styleFocus, row, width))
		} else {
			lines = append(lines, fitWidth(row, width))
		}
	}
	return padPane(lines, width, height)
}

func (m Model) renderFilesPane(width, height int) string {
	lines := m.renderPaneHeader(width, m.mainTitle())
	lines = append(lines, cell(styleHeader,
		fmt.Sprintf("%-4s %6s %6s %s", "ST", "+", "-", "PATH"),
		width,
	))

	h := height - 2
	if h < 1 {
		h = 1
	}
	end := min(len(m.files), m.fileOffset+h)
	for i := m.fileOffset; i < end; i++ {
		row := formatFileRow(m.files[i])
		if i == m.fileCursor {
			lines = append(lines, cell(styleFocus, row, width))
		} else {
			lines = append(lines, fitWidth(row, width))
		}
	}
	if len(m.files) == 0 {
		lines = append(lines, fitWidth(styleMuted.Render("(no files)"), width))
	}
	return padPane(lines, width, height)
}

func (m Model) renderHistoryPane(width, height int) string {
	title := m.mainTitle()
	if m.historyPath != "" {
		title = " history · " + m.historyPath
	}
	lines := m.renderPaneHeader(width, title)
	lines = append(lines, cell(styleHeader,
		fmt.Sprintf("%-7s %-10s %-12s %s", "HASH", "DATE", "AUTHOR", "SUBJECT"),
		width,
	))

	h := height - 2
	if h < 1 {
		h = 1
	}
	end := min(len(m.history), m.historyOffset+h)
	for i := m.historyOffset; i < end; i++ {
		row := formatCommitRow(m.history[i])
		if i == m.historyCursor {
			lines = append(lines, cell(styleFocus, row, width))
		} else {
			lines = append(lines, fitWidth(row, width))
		}
	}
	if m.loadingHistory {
		lines = append(lines, fitWidth(styleMuted.Render(" loading…"), width))
	} else if len(m.history) == 0 {
		lines = append(lines, fitWidth(styleMuted.Render("(no history)"), width))
	}
	return padPane(lines, width, height)
}

func (m Model) renderBlamePane(width, height int) string {
	title := " blame"
	if m.blamePath != "" {
		title = fmt.Sprintf(" blame · %s @ %s", m.blamePath, shortHash(m.blameRev))
	}
	lines := m.renderPaneHeader(width, title)
	lines = append(lines, cell(styleHeader,
		fmt.Sprintf("%4s %-7s %-10s %-10s %s", "LINE", "COMMIT", "AGE", "AUTHOR", "CODE"),
		width,
	))

	h := height - 2
	if h < 1 {
		h = 1
	}
	newest, oldest := blameAgeRange(m.blame)
	end := min(len(m.blame), m.blameOffset+h)
	for i := m.blameOffset; i < end; i++ {
		row := formatBlameRow(m.blame[i])
		if i == m.blameCursor {
			lines = append(lines, cell(styleFocus, row, width))
		} else {
			lines = append(lines, cell(blameAgeStyle(m.blame[i].When, newest, oldest), row, width))
		}
	}
	if m.loadingBlame {
		lines = append(lines, fitWidth(styleMuted.Render(" loading…"), width))
	} else if len(m.blame) == 0 {
		lines = append(lines, fitWidth(styleMuted.Render("(no blame)"), width))
	}
	return padPane(lines, width, height)
}

func padPane(lines []string, width, height int) string {
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines[:height], "\n")
}

func formatCommitRow(c git.Commit) string {
	date := c.Date.Local().Format("2006-01-02")
	author := c.Author
	if runewidth.StringWidth(author) > 12 {
		author = runewidth.Truncate(author, 12, "…")
	}
	return fmt.Sprintf("%-7s %s %-12s %s", c.ShortHash, date, author, c.Subject)
}

func formatFileRow(f git.FileChange) string {
	st := f.Status
	if st == "" {
		st = "M"
	}
	path := f.Path
	if f.OldPath != "" {
		path = f.OldPath + " → " + f.Path
	}
	return fmt.Sprintf("%-4s +%-5d -%-5d %s", st, f.Additions, f.Deletions, path)
}

func formatBlameRow(b git.BlameLine) string {
	author := b.Author
	if runewidth.StringWidth(author) > 10 {
		author = runewidth.Truncate(author, 10, "…")
	}
	age := relativeAge(b.When)
	code := strings.ReplaceAll(b.Text, "\t", "    ")
	return fmt.Sprintf("%4d %-7s %-10s %-10s %s", b.Line, b.ShortHash, age, author, code)
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
	var lines []string
	if m.focus == FocusDetail {
		lines = append(lines, cell(styleFocus, " detail", width))
	} else {
		lines = append(lines, cell(styleMuted, " detail", width))
	}

	body := m.detailLines()
	h := height - 1
	if h < 1 {
		h = 1
	}
	if m.loadingDetail && m.detail == nil {
		body = []string{styleMuted.Render(" loading…")}
	}
	end := min(len(body), m.detailOffset+h)
	for i := m.detailOffset; i < end; i++ {
		lines = append(lines, renderDetailLine(body[i], width))
	}
	return padPane(lines, width, height)
}

func (m Model) detailLines() []string {
	if m.detail == nil {
		return []string{styleMuted.Render("(no commit selected)")}
	}
	d := m.detail
	var out []string
	if m.main == MainBlame && len(m.blame) > 0 {
		bl := m.blame[m.blameCursor]
		out = append(out, styleMuted.Render(fmt.Sprintf("line %d · %s", bl.Line, relativeAge(bl.When))))
		if bl.Summary != "" {
			out = append(out, styleTitle.Render(bl.Summary))
		}
		out = append(out, "")
	}
	out = append(out, styleHash.Render(d.Commit.Hash))
	out = append(out, fmt.Sprintf("%s <%s>", d.Commit.Author, d.Commit.Email))
	out = append(out, d.Commit.Date.Local().Format(time.RFC1123))
	if m.detailFilterPath != "" {
		out = append(out, styleMuted.Render("path  "+m.detailFilterPath))
	}
	out = append(out, "")
	out = append(out, styleTitle.Render(d.Commit.Subject))
	if d.Body != "" {
		out = append(out, "")
		out = append(out, strings.Split(d.Body, "\n")...)
	}
	out = append(out, "")
	out = append(out, fmt.Sprintf("%d files  %s  %s",
		d.Commit.Files,
		styleAdd.Render(fmt.Sprintf("+%d", d.Commit.Additions)),
		styleDel.Render(fmt.Sprintf("-%d", d.Commit.Deletions)),
	))
	if stat := strings.TrimRight(d.Stat, "\n"); stat != "" {
		for _, line := range strings.Split(stat, "\n") {
			out = append(out, styleMuted.Render(line))
		}
	}
	out = append(out, "")
	if diff := strings.TrimRight(d.Diff, "\n"); diff != "" {
		out = append(out, strings.Split(diff, "\n")...)
	}
	return out
}

func renderDetailLine(line string, width int) string {
	if strings.Contains(line, "\x1b[") {
		return fitWidth(line, width)
	}
	switch {
	case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
		return cell(styleAddWash, line, width)
	case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
		return cell(styleDelWash, line, width)
	default:
		return fitWidth(line, width)
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
