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
	FocusCommits Focus = iota
	FocusDetail
)

// Model is the root TUI state.
type Model struct {
	repo *git.Repo

	width  int
	height int
	focus  Focus

	commits       []git.Commit
	cursor        int
	commitOffset  int
	detail        *git.CommitDetail
	detailOffset  int
	loading       bool
	loadingDetail bool
	err           string
	status        string

	branch string
	head   string
}

// New builds a model bound to repo. Call Init via the Bubble Tea program.
func New(repo *git.Repo) Model {
	return Model{
		repo:    repo,
		focus:   FocusCommits,
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
	detail git.CommitDetail
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

func loadDetailCmd(repo *git.Repo, hash string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		detail, err := repo.Show(ctx, hash)
		return detailLoadedMsg{hash: hash, detail: detail, err: err}
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
		m.status = fmt.Sprintf("%d commits", len(m.commits))
		if len(m.commits) > 0 {
			m.loadingDetail = true
			return m, loadDetailCmd(m.repo, m.commits[0].Hash)
		}
		return m, nil

	case detailLoadedMsg:
		m.loadingDetail = false
		if len(m.commits) == 0 || m.commits[m.cursor].Hash != msg.hash {
			// Stale response from a previous selection.
			return m, nil
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
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		if m.focus == FocusCommits {
			m.focus = FocusDetail
		} else {
			m.focus = FocusCommits
		}
		return m, nil
	case "?":
		m.status = "j/k move · tab focus · enter reload detail · q quit · archaeology-only (read-only)"
		return m, nil
	}

	switch m.focus {
	case FocusCommits:
		return m.handleCommitKeys(msg)
	case FocusDetail:
		return m.handleDetailKeys(msg)
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
		// Single "g" jumps to top for MVP; gg chord comes with the key registry.
		m.cursor = 0
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "G", "end":
		m.cursor = len(m.commits) - 1
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "ctrl+d":
		m.cursor = min(len(m.commits)-1, m.cursor+m.commitPage())
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "ctrl+u":
		m.cursor = max(0, m.cursor-m.commitPage())
		m.ensureCommitVisible()
		return m, m.reloadDetail()
	case "enter":
		return m, m.reloadDetail()
	}
	return m, nil
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
	}
	return m, nil
}

func (m *Model) reloadDetail() tea.Cmd {
	if len(m.commits) == 0 {
		return nil
	}
	m.loadingDetail = true
	m.detailOffset = 0
	return loadDetailCmd(m.repo, m.commits[m.cursor].Hash)
}

func (m *Model) ensureCommitVisible() {
	h := m.commitViewHeight()
	if h <= 0 {
		return
	}
	if m.cursor < m.commitOffset {
		m.commitOffset = m.cursor
	}
	if m.cursor >= m.commitOffset+h {
		m.commitOffset = m.cursor - h + 1
	}
}

func (m Model) commitPage() int {
	return max(1, m.commitViewHeight()-1)
}

func (m Model) commitViewHeight() int {
	return max(1, m.bodyHeight()-2) // account for header row
}

func (m Model) detailViewHeight() int {
	return max(1, m.bodyHeight())
}

func (m Model) bodyHeight() int {
	// title + status
	return max(1, m.height-3)
}

func (m Model) commitPaneWidth() int {
	if m.width < 80 {
		return max(20, m.width)
	}
	return max(40, m.width*55/100)
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
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("180"))
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleFocus   = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("236"))
	styleHeader  = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true)
	styleAdd     = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	styleDel     = lipgloss.NewStyle().Foreground(lipgloss.Color("174"))
	styleErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleHash    = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
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
	if m.err != "" {
		msg = styleErr.Render(m.err)
	} else if m.loading || m.loadingDetail {
		msg = styleMuted.Render(m.status + " · fetching…")
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
		return m.renderCommitPane(m.width, h)
	}
	cw := m.commitPaneWidth()
	dw := m.width - cw - 1
	left := m.renderCommitPane(cw, h)
	right := m.renderDetailPane(dw, h)
	sep := styleMuted.Width(1).Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

func (m Model) renderCommitPane(width, height int) string {
	var lines []string
	if m.focus == FocusCommits {
		lines = append(lines, cell(styleFocus, " commits", width))
	} else {
		lines = append(lines, cell(styleMuted, " commits", width))
	}
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
		lines = append(lines, fitWidth(colorDiffLine(body[i]), width))
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines[:height], "\n")
}

func (m Model) detailLines() []string {
	if m.detail == nil {
		return []string{styleMuted.Render("(no commit selected)")}
	}
	d := m.detail
	var out []string
	out = append(out, styleHash.Render(d.Commit.Hash))
	out = append(out, fmt.Sprintf("%s <%s>", d.Commit.Author, d.Commit.Email))
	out = append(out, d.Commit.Date.Local().Format(time.RFC1123))
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
	// One grid row per physical line — embedded newlines desync JoinHorizontal.
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

func colorDiffLine(line string) string {
	// Only colour plain diff text; styled meta lines are left as-is.
	if strings.Contains(line, "\x1b[") {
		return line
	}
	switch {
	case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
		return styleAdd.Render(line)
	case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
		return styleDel.Render(line)
	default:
		return line
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

// fitWidth truncates/pads to a visual cell width, preserving ANSI resets.
func fitWidth(s string, width int) string {
	return cell(lipgloss.NewStyle(), s, width)
}

// clampFrame forces the view to exactly height rows so the alt screen never scrolls.
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
