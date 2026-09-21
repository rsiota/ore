package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// dagJumpMsg is the result of an async soft DAG hop (child / merge-base).
type dagJumpMsg struct {
	kind   string // "child" | "merge-base"
	from   string
	to     string
	total  int // candidates (children count, or 1 for merge-base)
	err    error
}

func (m Model) selectedCommitParents() []string {
	switch m.main {
	case MainCommits:
		idx := m.commitIndices()
		if m.cursor < 0 || m.cursor >= len(idx) {
			return nil
		}
		return m.commits[idx[m.cursor]].Parents
	case MainHistory:
		if pc, ok := m.selectedHistory(); ok {
			return pc.Parents
		}
	case MainBlame:
		if bl, ok := m.selectedBlameLine(); ok && bl.Hash != "" {
			// Prefer parents from the commit log when the blamed commit is loaded.
			for _, c := range m.commits {
				if hashMatch(c.Hash, bl.Hash) {
					return c.Parents
				}
			}
		}
	case MainPickaxe:
		if h, ok := m.selectedPickaxeHit(); ok {
			return h.Commit.Parents
		}
	}
	return nil
}

func (m Model) dagSourceHash() string {
	switch m.main {
	case MainCommits, MainHistory, MainPickaxe:
		return m.selectedHash()
	case MainBlame:
		if bl, ok := m.selectedBlameLine(); ok {
			return bl.Hash
		}
	}
	return ""
}

func (m Model) dagTipRev() string {
	if m.viewRev != "" {
		return m.viewRev
	}
	if m.head != "" {
		return m.head
	}
	return "HEAD"
}

// jumpDagParent moves to the first parent of the current commit (soft DAG hop).
func (m Model) jumpDagParent() (tea.Model, tea.Cmd) {
	parents := m.selectedCommitParents()
	if len(parents) == 0 {
		m.status = "dag · no parent"
		return m, nil
	}
	target := parents[0]
	note := ""
	if len(parents) > 1 {
		note = fmt.Sprintf(" · merge 1/%d", len(parents))
	}
	cmd := (&m).applyDagJump("parent", target, len(parents), note)
	return m, cmd
}

// jumpDagChild loads direct children and jumps to the first one.
func (m Model) jumpDagChild() (tea.Model, tea.Cmd) {
	from := m.dagSourceHash()
	if from == "" || m.repo == nil {
		m.status = "dag · no commit"
		return m, nil
	}
	m.status = "dag · finding children…"
	repo := m.repo
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		kids, err := repo.ChildrenOf(ctx, from)
		if err != nil {
			return dagJumpMsg{kind: "child", from: from, err: err}
		}
		if len(kids) == 0 {
			return dagJumpMsg{kind: "child", from: from, total: 0}
		}
		return dagJumpMsg{kind: "child", from: from, to: kids[0], total: len(kids)}
	}
}

// jumpDagMergeBase jumps to merge-base(tip, current).
func (m Model) jumpDagMergeBase() (tea.Model, tea.Cmd) {
	from := m.dagSourceHash()
	if from == "" || m.repo == nil {
		m.status = "dag · no commit"
		return m, nil
	}
	tip := m.dagTipRev()
	m.status = fmt.Sprintf("dag · merge-base with %s…", shortHash(tip))
	repo := m.repo
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		base, err := repo.MergeBase(ctx, tip, from)
		if err != nil {
			return dagJumpMsg{kind: "merge-base", from: from, err: err}
		}
		return dagJumpMsg{kind: "merge-base", from: from, to: base, total: 1}
	}
}

func (m Model) handleDagJumpMsg(msg dagJumpMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "dag · " + msg.err.Error()
		return m, nil
	}
	switch msg.kind {
	case "child":
		if msg.total == 0 || msg.to == "" {
			m.status = "dag · no children"
			return m, nil
		}
		note := ""
		if msg.total > 1 {
			note = fmt.Sprintf(" · 1/%d", msg.total)
		}
		cmd := (&m).applyDagJump("child", msg.to, msg.total, note)
		return m, cmd
	case "merge-base":
		if msg.to == "" {
			m.status = "dag · no merge-base"
			return m, nil
		}
		if hashMatch(msg.to, msg.from) {
			m.status = "dag · already at merge-base"
			return m, nil
		}
		cmd := (&m).applyDagJump("merge-base", msg.to, 1, "")
		return m, cmd
	default:
		return m, nil
	}
}

// applyDagJump moves the cursor to hash when present in the current/commit
// list; otherwise opens detail for that hash on the commit grid.
func (m *Model) applyDagJump(kind, hash string, _ int, note string) tea.Cmd {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		m.status = "dag · empty target"
		return nil
	}

	label := kind
	switch kind {
	case "parent":
		label = "parent"
	case "child":
		label = "child"
	case "merge-base":
		label = "merge-base"
	}

	// Prefer staying in history when the hop is on the followed path.
	if m.main == MainHistory && m.selectHistoryCommitOK(hash) {
		m.status = fmt.Sprintf("dag · %s %s%s", label, shortHash(hash), note)
		return m.reloadDetail()
	}

	m.filter = ""
	m.filterTyping = false
	if m.jumpToCommit(hash) {
		m.status = fmt.Sprintf("dag · %s %s%s", label, shortHash(hash), note)
		return m.reloadDetail()
	}

	// Not in the loaded window — still show the commit in detail.
	m.main = MainCommits
	m.focus = FocusMain
	m.detailFilterPath = ""
	m.detailExpectHash = hash
	m.loadingDetail = true
	m.status = fmt.Sprintf("dag · %s %s%s · not in log window", label, shortHash(hash), note)
	return m.reloadDetailNow()
}

func (m *Model) selectHistoryCommitOK(hash string) bool {
	if hash == "" {
		return false
	}
	idx := m.historyIndices()
	for i, src := range idx {
		if hashMatch(m.history[src].Hash, hash) {
			m.historyCursor = i
			m.ensureHistoryVisible()
			return true
		}
	}
	return false
}
