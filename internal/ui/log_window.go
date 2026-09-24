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
	if cmd := m.loadMore(extra); cmd != nil {
		return cmd
	}
	if m.logCapped() {
		return nil
	}
	m.status = "already showing all commits"
	return nil
}
