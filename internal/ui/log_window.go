package ui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func (m Model) effectiveLogLimit() int {
	if m.logLimit > 0 {
		return m.logLimit
	}
	return git.DefaultLogLimit
}

func (m Model) effectiveHistoryLimit() int {
	if m.historyLimit > 0 {
		return m.historyLimit
	}
	return git.DefaultLogLimit
}

func (m Model) defaultHitLimit() int {
	switch m.pickKind {
	case hitListCouple:
		return git.CoChangeCommitCap
	case hitListAuthors:
		return git.OwnershipCommitCap
	default:
		return git.DefaultPickaxeLimit
	}
}

func (m Model) effectiveHitLimit() int {
	if m.pickLimit > 0 {
		return m.pickLimit
	}
	return m.defaultHitLimit()
}

// logCapped reports whether the loaded commit window is shorter than the tip.
func (m Model) logCapped() bool {
	if len(m.commits) == 0 {
		return false
	}
	if m.logTotal > 0 {
		return len(m.commits) < m.logTotal
	}
	return len(m.commits) >= m.effectiveLogLimit()
}

func (m Model) historyCapped() bool {
	return len(m.history) > 0 && len(m.history) >= m.effectiveHistoryLimit()
}

func (m Model) hitsCapped() bool {
	return len(m.pickaxe) > 0 && len(m.pickaxe) >= m.effectiveHitLimit()
}

func (m Model) commitsWindowText() string {
	n := len(m.commits)
	if m.logTotal > n && m.logTotal > 0 {
		return fmt.Sprintf("%d/%d commits", n, m.logTotal)
	}
	if m.logTotal == 0 && n >= m.effectiveLogLimit() {
		return fmt.Sprintf("%d+ commits", n)
	}
	return fmt.Sprintf("%d commits", n)
}

func (m Model) commitsTitle() string {
	n := len(m.commits)
	if m.logTotal > n && m.logTotal > 0 {
		return fmt.Sprintf("commits · %d/%d", n, m.logTotal)
	}
	if m.logTotal == 0 && n >= m.effectiveLogLimit() && n > 0 {
		return fmt.Sprintf("commits · %d+", n)
	}
	return "commits"
}

func cappedSuffix(n, limit int) string {
	if limit > 0 && n >= limit {
		return " · capped"
	}
	return ""
}

func (m *Model) loadMore(extra int) tea.Cmd {
	if m.loading || m.repo == nil || !m.logCapped() {
		return nil
	}
	if extra <= 0 {
		extra = git.DefaultLogLimit
	}
	m.logLimit = m.effectiveLogLimit() + extra
	m.refreshPreferHash = m.commitListHash()
	m.loading = true
	m.logExtending = true
	m.err = ""
	m.status = "loading older commits…"
	return loadCommitsCmd(m.repo, m.viewRev, m.logLimit)
}

func (m *Model) loadMoreHistory(extra int) tea.Cmd {
	if m.loadingHistory || m.repo == nil || m.historyPath == "" || !m.historyCapped() {
		return nil
	}
	if extra <= 0 {
		extra = git.DefaultLogLimit
	}
	if h, ok := m.selectedHistory(); ok {
		m.historyPreferHash = h.Hash
	}
	m.historyLimit = m.effectiveHistoryLimit() + extra
	m.historyExtending = true
	m.loadingHistory = true
	m.err = ""
	m.status = "loading older history…"
	return loadHistoryCmd(m.repo, m.historyPath, m.historyLimit)
}

func (m *Model) loadMoreHits(extra int) tea.Cmd {
	if m.loadingPick || m.repo == nil || !m.hitsCapped() {
		return nil
	}
	if extra <= 0 {
		extra = m.defaultHitLimit()
	}
	if h, ok := m.selectedPickaxeHit(); ok {
		m.pickPreferHash = h.Commit.Hash
	}
	m.pickLimit = m.effectiveHitLimit() + extra
	m.pickExtending = true
	m.loadingPick = true
	m.err = ""
	m.status = "loading older hits…"
	switch m.pickKind {
	case hitListCouple:
		return loadCoupleCmd(m.repo, m.coupleSeeds, m.couplePartner, m.pickLimit)
	case hitListAuthors:
		return loadAuthorCmd(m.repo, m.authorFilter, m.authorRev, m.authorPaths, m.pickLimit)
	default:
		return loadPickaxeCmd(m.repo, git.PickaxeOptions{
			Query:    m.pickQuery,
			Mode:     m.pickMode,
			Path:     m.pickPath,
			Rev:      m.viewRev,
			MaxCount: m.pickLimit,
		})
	}
}

func (m *Model) requestHistory(path string) tea.Cmd {
	m.historyPath = path
	m.loadingHistory = true
	if !m.historyExtending && !m.restoreAfterHistory {
		m.historyLimit = 0
	}
	return loadHistoryCmd(m.repo, path, m.effectiveHistoryLimit())
}

func (m *Model) exMore(args []string) tea.Cmd {
	extra := 0
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			m.status = "E488: :more [count]"
			return nil
		}
		extra = n
	}
	switch m.main {
	case MainHistory:
		if cmd := m.loadMoreHistory(extra); cmd != nil {
			return cmd
		}
		if m.historyCapped() {
			return nil
		}
		m.status = "already showing all history"
		return nil
	case MainPickaxe:
		if cmd := m.loadMoreHits(extra); cmd != nil {
			return cmd
		}
		if m.hitsCapped() {
			return nil
		}
		m.status = "already showing all hits"
		return nil
	}
	if cmd := m.loadMore(extra); cmd != nil {
		return cmd
	}
	if m.logCapped() {
		return nil
	}
	m.status = "already showing all commits"
	return nil
}
