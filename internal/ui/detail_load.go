package ui

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

// Detail fetch tuning — keep scroll snappy while still using git CLI.
const (
	detailDebounce     = 75 * time.Millisecond
	detailLoadTimeout  = 30 * time.Second
	maxDiffBytes       = 512 << 10 // soft-cap raw patch payload (~512KiB)
	maxDetailBodyLines = 6000     // soft-cap logical detail rows after parse
)

type detailDebounceMsg struct {
	seq int
}

type detailHeaderMsg struct {
	seq    int
	hash   string
	path   string
	detail git.CommitDetail
	err    error
}

type detailPatchMsg struct {
	seq       int
	hash      string
	path      string
	diff      string
	truncated bool
	err       error
}

func (m *Model) cancelDetailLoad() {
	if m.detailCancel != nil {
		m.detailCancel()
		m.detailCancel = nil
	}
	m.detailCtx = nil
}

// reloadDetail schedules a debounced, cancellable header→patch load for the
// current selection. Rapid cursor motion only pays for the final commit.
func (m *Model) reloadDetail() tea.Cmd {
	hash := m.selectedHash()
	if m.detailExpectHash != "" {
		hash = m.detailExpectHash
	}
	if hash == "" {
		return nil
	}
	path := m.detailPathForView()
	m.detailFilterPath = path
	m.loadingDetail = true
	m.detailOffset = 0
	m.detailDebounceSeq++
	seq := m.detailDebounceSeq
	// Drop any in-flight git work immediately so scroll doesn't pile up.
	m.cancelDetailLoad()
	return tea.Tick(detailDebounce, func(time.Time) tea.Msg {
		return detailDebounceMsg{seq: seq}
	})
}

// reloadDetailNow skips the debounce (explicit jumps / open-files).
func (m *Model) reloadDetailNow() tea.Cmd {
	hash := m.selectedHash()
	if m.detailExpectHash != "" {
		hash = m.detailExpectHash
	}
	if hash == "" {
		return nil
	}
	m.detailFilterPath = m.detailPathForView()
	m.loadingDetail = true
	m.detailOffset = 0
	m.detailDebounceSeq++ // invalidate pending ticks
	return m.beginDetailLoad()
}

func (m *Model) detailPathForView() string {
	switch m.main {
	case MainFiles:
		idx := m.fileIndices()
		if m.fileCursor >= 0 && m.fileCursor < len(idx) {
			return m.files[idx[m.fileCursor]].Path
		}
	case MainHistory:
		return m.historyPath
	case MainBlame:
		return m.blamePath
	}
	return ""
}

func (m *Model) beginDetailLoad() tea.Cmd {
	hash := m.selectedHash()
	if m.detailExpectHash != "" {
		hash = m.detailExpectHash
	}
	if hash == "" {
		m.loadingDetail = false
		return nil
	}
	path := m.detailFilterPath
	m.cancelDetailLoad()
	ctx, cancel := context.WithTimeout(context.Background(), detailLoadTimeout)
	m.detailCtx = ctx
	m.detailCancel = cancel
	m.detailLoadSeq++
	seq := m.detailLoadSeq
	repo := m.repo
	m.loadingDetail = true
	m.detailPatchPending = true
	m.detailPending = nil
	// Keep showing the previous detail until header+patch are both ready.

	return func() tea.Msg {
		detail, err := repo.ShowHeader(ctx, hash, path)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return detailHeaderMsg{seq: seq, hash: hash, path: path, detail: detail, err: err}
	}
}

func (m *Model) continueDetailPatch(hash, path string) tea.Cmd {
	seq := m.detailLoadSeq
	ctx := m.detailCtx
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), detailLoadTimeout)
		m.detailCtx = ctx
		m.detailCancel = cancel
	}
	unified := m.zenContext
	repo := m.repo
	return func() tea.Msg {
		diff, err := repo.ShowPatch(ctx, hash, path, unified)
		if ctx.Err() != nil {
			return detailPatchMsg{seq: seq, hash: hash, path: path, err: ctx.Err()}
		}
		truncated := false
		if err == nil {
			diff, truncated = truncateDiff(diff)
		}
		return detailPatchMsg{seq: seq, hash: hash, path: path, diff: diff, truncated: truncated, err: err}
	}
}

func truncateDiff(diff string) (string, bool) {
	if len(diff) <= maxDiffBytes {
		return diff, false
	}
	cut := maxDiffBytes
	if i := lastIndexByte(diff[:cut], '\n'); i > maxDiffBytes/2 {
		cut = i + 1
	}
	return diff[:cut], true
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func (m *Model) handleDetailDebounce(msg detailDebounceMsg) (tea.Model, tea.Cmd) {
	if msg.seq != m.detailDebounceSeq {
		return *m, nil
	}
	return *m, m.beginDetailLoad()
}

func (m *Model) handleDetailHeader(msg detailHeaderMsg) (tea.Model, tea.Cmd) {
	if msg.seq != m.detailLoadSeq {
		return *m, nil
	}
	expect := m.selectedHash()
	if m.detailExpectHash != "" {
		expect = m.detailExpectHash
	}
	if !hashMatch(msg.hash, expect) || msg.path != m.detailFilterPath {
		return *m, nil
	}
	if msg.err != nil {
		if errors.Is(msg.err, context.Canceled) {
			return *m, nil
		}
		m.err = msg.err.Error()
		m.detail = nil
		m.detailPending = nil
		m.detailPath = ""
		m.loadingDetail = false
		m.detailPatchPending = false
		m.invalidateDetailCache()
		return *m, nil
	}
	m.err = ""
	d := msg.detail
	// Stage header only — keep painting the previous detail to avoid flicker.
	pending := d
	m.detailPending = &pending
	m.detailPatchPending = true
	if msg.path == "" && m.main == MainFiles && hashMatch(d.Commit.Hash, m.filesCommitHash) {
		m.syncFilesFrom(d.Files, d.Commit.Hash)
	}
	if m.openFilesPending && m.main == MainCommits {
		m.openFilesPending = false
		m.enterFilesViewFrom(d)
		// Files view needs path-scoped detail; restart load for selected file.
		return *m, m.reloadDetailNow()
	}
	return *m, m.continueDetailPatch(msg.hash, msg.path)
}

func (m *Model) handleDetailPatch(msg detailPatchMsg) (tea.Model, tea.Cmd) {
	if msg.seq != m.detailLoadSeq {
		return *m, nil
	}
	expect := m.selectedHash()
	if m.detailExpectHash != "" {
		expect = m.detailExpectHash
	}
	if !hashMatch(msg.hash, expect) || msg.path != m.detailFilterPath {
		return *m, nil
	}
	if msg.err != nil {
		if errors.Is(msg.err, context.Canceled) {
			return *m, nil
		}
		m.err = msg.err.Error()
		m.loadingDetail = false
		m.detailPatchPending = false
		m.detailPending = nil
		return *m, nil
	}
	if m.detailPending == nil || !hashMatch(m.detailPending.Commit.Hash, msg.hash) {
		return *m, nil
	}
	d := *m.detailPending
	d.Diff = msg.diff
	m.detail = &d
	m.detailPending = nil
	m.detailPath = msg.path
	m.detailTruncated = msg.truncated
	m.detailOffset = 0
	m.detailPatchPending = false
	m.loadingDetail = false
	m.detailExpectHash = ""
	m.invalidateDetailCache()
	return *m, nil
}
