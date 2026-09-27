package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

const (
	minSidebarWidth      = 16
	defaultSidebarWidth  = 24
	minMainWithSidebar   = 28
	minRightWithSidebar  = 22
	maxTreeFilterMatches = 500
)

type treeLoadedMsg struct {
	rev    string
	prefix string
	kids   []git.TreeEntry
	err    error
}

type treeFilesLoadedMsg struct {
	rev   string
	query string
	files []string
	err   error
}

func (m Model) sidebarDocked() bool {
	return m.sidebarOpen && m.width >= 80
}

func (m Model) sidebarNarrow() bool {
	return m.sidebarOpen && m.width < 80 && m.focus == FocusSidebar && !m.explorer.Opened()
}

func (m Model) sidebarShowing() bool {
	return m.sidebarDocked() || m.sidebarNarrow()
}

func (m Model) treeRev() string {
	if m.viewRev != "" {
		return m.viewRev
	}
	return "HEAD"
}

func (m Model) sidebarWidth() int {
	if !m.sidebarDocked() {
		return 0
	}
	w := m.sidebarSplitW
	if w <= 0 {
		w = defaultSidebarWidth
	}
	return m.clampSidebarWidth(w)
}

func (m Model) clampSidebarWidth(w int) int {
	maxSide := m.width - minMainWithSidebar - minRightWithSidebar
	if maxSide < minSidebarWidth {
		maxSide = minSidebarWidth
	}
	if w < minSidebarWidth {
		w = minSidebarWidth
	}
	if w > maxSide {
		w = maxSide
	}
	return w
}

func (m *Model) toggleSidebar() tea.Cmd {
	m.chordG = false
	if m.sidebarOpen {
		m.sidebarOpen = false
		if m.focus == FocusSidebar {
			m.focus = FocusMain
		}
		m.refreshStatus()
		m.invalidateDetailCache()
		return nil
	}
	m.sidebarOpen = true
	m.focus = FocusSidebar
	m.status = m.treeStatus()
	m.invalidateDetailCache()
	return m.ensureTreeLoaded()
}

func (m *Model) ensureTreeLoaded() tea.Cmd {
	rev := m.treeRev()
	if m.tree.rev != rev || m.tree.kids == nil {
		prefer := m.tree.preferPath
		if prefer == "" {
			prefer = m.tree.selectedPath()
		}
		m.tree.reset(rev, prefer)
	}
	var cmds []tea.Cmd
	for _, prefix := range m.tree.missingPrefixes(m.tree.preferPath) {
		cmds = append(cmds, m.requestTree(prefix))
	}
	if len(cmds) == 0 {
		if _, ok := m.tree.kids[""]; !ok {
			cmds = append(cmds, m.requestTree(""))
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

func (m *Model) requestTree(prefix string) tea.Cmd {
	if m.repo == nil {
		return nil
	}
	m.loadingTree = true
	rev := m.treeRev()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		kids, err := m.repo.ListTree(ctx, rev, prefix)
		return treeLoadedMsg{rev: rev, prefix: prefix, kids: kids, err: err}
	}
}

func (m *Model) ensureTreeFiles() tea.Cmd {
	if m.tree.filesLoaded {
		return nil
	}
	return m.requestTreeFiles()
}

func (m *Model) requestTreeFiles() tea.Cmd {
	if m.repo == nil {
		return nil
	}
	m.loadingTree = true
	rev := m.treeRev()
	query := m.tree.filter
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		files, err := m.repo.ListFiles(ctx, rev, "")
		return treeFilesLoadedMsg{rev: rev, query: query, files: files, err: err}
	}
}

func (m Model) handleTreeLoaded(msg treeLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.rev != m.treeRev() {
		return m, nil
	}
	m.loadingTree = false
	if msg.err != nil {
		m.err = msg.err.Error()
		m.status = "tree error"
		return m, nil
	}
	if m.tree.kids == nil {
		m.tree.reset(msg.rev, m.tree.preferPath)
	}
	m.tree.rev = msg.rev
	m.tree.applyKids(msg.prefix, msg.kids)
	if m.tree.preferPath != "" {
		if missing := m.tree.missingPrefixes(m.tree.preferPath); len(missing) > 0 {
			m.loadingTree = true
			return m, m.requestTree(missing[0])
		}
		m.tree.selectPath(m.tree.preferPath)
		m.tree.preferPath = ""
	}
	m.tree.ensureVisible(m.sidebarListHeight())
	if m.focus == FocusSidebar {
		m.status = m.treeStatus()
	}
	return m, nil
}

func (m Model) handleTreeFilesLoaded(msg treeFilesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.rev != m.treeRev() {
		return m, nil
	}
	m.loadingTree = false
	if msg.err != nil {
		m.err = msg.err.Error()
		m.status = "tree error"
		return m, nil
	}
	if strings.TrimSpace(m.tree.filter) == "" {
		return m, nil
	}
	m.tree.applyMatches(msg.files)
	m.tree.ensureVisible(m.sidebarListHeight())
	if m.focus == FocusSidebar {
		m.status = m.treeStatus()
	}
	return m, nil
}

func (m Model) handleSidebarKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	h := m.sidebarListHeight()
	switch msg.String() {
	case "j", "down":
		m.tree.move(1)
	case "k", "up":
		m.tree.move(-1)
	case "g", "home":
		m.tree.gotoTop()
	case "G", "end":
		m.tree.gotoBottom()
	case "ctrl+d":
		m.tree.move(max(1, h/2))
	case "ctrl+u":
		m.tree.move(-max(1, h/2))
	case "l", "right":
		return m.sidebarExpandOrOpen()
	case "h", "left", "backspace":
		if !m.tree.collapseOrParent() {
			m.focus = FocusMain
			m.refreshStatus()
			return m, nil
		}
	case "enter":
		return m.sidebarActivate(false)
	case "b":
		return m.sidebarActivate(true)
	default:
		return m, nil
	}
	m.tree.ensureVisible(h)
	m.status = m.treeStatus()
	return m, nil
}

func (m Model) sidebarExpandOrOpen() (tea.Model, tea.Cmd) {
	row, ok := m.tree.selected()
	if !ok {
		return m, nil
	}
	if row.dir {
		prefix, changed := m.tree.toggleExpand()
		if prefix != "" {
			m.status = "loading " + prefix + "…"
			return m, m.requestTree(prefix)
		}
		if changed {
			m.tree.ensureVisible(m.sidebarListHeight())
			m.status = m.treeStatus()
		}
		return m, nil
	}
	return m.sidebarActivate(false)
}

func (m Model) sidebarActivate(blame bool) (tea.Model, tea.Cmd) {
	row, ok := m.tree.selected()
	if !ok {
		return m, nil
	}
	if row.dir {
		return m.sidebarExpandOrOpen()
	}
	m.focus = FocusMain
	if blame {
		rev := m.treeRev()
		m.status = fmt.Sprintf("loading blame · %s @ %s", row.path, shortHash(rev))
		return m, m.startBlame(row.path, rev, MainFiles)
	}
	m.historyCol = histColHash
	m.historySortCol = -1
	m.historySortDir = SortNone
	m.status = fmt.Sprintf("loading history · %s", row.path)
	return m, m.requestHistory(row.path)
}

func (m Model) handleTreeFilterKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.tree.filter = ""
		m.tree.filterTyping = false
		m.tree.clampCursor()
		m.tree.ensureVisible(m.sidebarListHeight())
		m.status = m.treeStatus()
		return m, nil
	case "enter":
		m.tree.filterTyping = false
		m.status = m.treeStatus()
		return m, nil
	case "backspace":
		if m.tree.filter != "" {
			r := []rune(m.tree.filter)
			m.tree.filter = string(r[:len(r)-1])
			m.tree.clampCursor()
		}
		m.status = m.treeFilterPrompt()
		return m, m.ensureTreeFiles()
	case "ctrl+u":
		m.tree.filter = ""
		m.tree.clampCursor()
		m.status = m.treeFilterPrompt()
		return m, nil
	default:
		if len(msg.Runes) > 0 && !msg.Alt && msg.Type == tea.KeyRunes {
			m.tree.filter += string(msg.Runes)
			m.tree.clampCursor()
			m.status = m.treeFilterPrompt()
			return m, m.ensureTreeFiles()
		}
		if s := msg.String(); len(s) == 1 && s[0] >= 32 {
			m.tree.filter += s
			m.tree.clampCursor()
			m.status = m.treeFilterPrompt()
			return m, m.ensureTreeFiles()
		}
	}
	return m, nil
}

func (m Model) treeFilterPrompt() string {
	return "filter tree: " + m.tree.filter + "█"
}

func (m Model) treeStatus() string {
	if row, ok := m.tree.selected(); ok {
		kind := "file"
		if row.dir {
			kind = "dir"
		}
		n := len(m.tree.rows())
		if strings.TrimSpace(m.tree.filter) != "" {
			return fmt.Sprintf("tree · %d matches · %s", n, row.path)
		}
		return fmt.Sprintf("tree · %s · %s", kind, row.path)
	}
	if m.loadingTree {
		return "tree · loading…"
	}
	return "tree"
}

func (m Model) sidebarListHeight() int {
	return max(1, m.paneContentHeight()-2)
}

func (m Model) renderSidebarPane(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	title := " tree"
	if m.viewRev != "" {
		title = " tree · " + m.viewRev
	} else if m.branch != "" {
		title = " tree · " + m.branch
	}
	if strings.TrimSpace(m.tree.filter) != "" {
		title += " · /" + m.tree.filter
	}
	rows := m.tree.rows()
	m.tree.ensureVisible(max(1, height-2))
	var lines []string
	lines = append(lines, cell(styleHeader, title, width))
	lines = append(lines, cell(styleGridBorder, strings.Repeat("─", width), width))
	innerH := max(1, height-2)
	end := min(len(rows), m.tree.offset+innerH)
	for i := m.tree.offset; i < end; i++ {
		lines = append(lines, m.renderTreeRow(rows[i], i == m.tree.cursor, width))
	}
	if len(rows) == 0 {
		msg := " loading…"
		if !m.loadingTree {
			msg = " empty tree"
		}
		if strings.TrimSpace(m.tree.filter) != "" {
			msg = " no matches"
		}
		lines = append(lines, fitWidth(styleMuted.Render(msg), width))
	}
	return padPane(lines, width, height)
}

func (m Model) renderTreeRow(row treeRow, selected bool, width int) string {
	indent := strings.Repeat("  ", row.depth)
	glyph := "  "
	label := row.name
	switch {
	case row.dir && row.expanded:
		glyph = "▾ "
	case row.dir:
		glyph = "▸ "
	}
	if strings.TrimSpace(m.tree.filter) != "" {
		label = row.path
	}
	text := indent + glyph + label
	if selected && m.focus == FocusSidebar {
		return cell(styleFocus, text, width)
	}
	if row.dir {
		return fitWidth(styleCell.Render(text), width)
	}
	return fitWidth(styleMuted.Render(text), width)
}

func (m Model) applySidebarClick(innerY int) Model {
	if m.focus == FocusDetail {
		m.leaveDetailYank()
	}
	if m.focus == FocusBlameYank {
		m.leaveBlameYank()
	}
	m.focus = FocusSidebar
	if innerY <= 1 {
		m.status = m.treeStatus()
		return m
	}
	row := m.tree.offset + innerY - 2
	n := len(m.tree.rows())
	if row >= 0 && row < n {
		m.tree.cursor = row
		m.tree.ensureVisible(m.sidebarListHeight())
	}
	m.status = m.treeStatus()
	return m
}
