package ui

import (
	"fmt"
	"strings"
)

const detailViewOverscan = 32

type detailRenderCache struct {
	key     string
	logical []string
	visual  []string // fully expanded when wrap is on
}

func (m *Model) invalidateDetailCache() {
	if m.detailCache == nil {
		m.detailCache = &detailRenderCache{}
		return
	}
	*m.detailCache = detailRenderCache{}
}

func (m Model) detailRenderKey(width int) string {
	hash := ""
	diffLen := 0
	if m.detail != nil {
		hash = m.detail.Commit.Hash
		diffLen = len(m.detail.Diff)
	}
	return fmt.Sprintf("%s|%s|%d|%d|%t|%d|%t|%d",
		hash, m.detailPath, m.diffMode, m.zenContext, m.detailWrap, width,
		m.detailTruncated, diffLen)
}

func (m Model) ensureDetailLogical(width int) []string {
	if m.detailCache == nil {
		m.detailCache = &detailRenderCache{}
	}
	key := m.detailRenderKey(width)
	if m.detailCache.key == key && m.detailCache.logical != nil {
		return m.detailCache.logical
	}
	body := m.detailLines()
	if m.detailTruncated && m.detail != nil && m.detail.Diff != "" {
		body = append(body, styleMuted.Render(strings.Repeat(" ", cellPad)+
			"… patch truncated for speed (path-filter or smaller commit for full)"))
	}
	body = clampDetailBody(body)
	m.detailCache.key = key
	m.detailCache.logical = body
	m.detailCache.visual = nil
	return body
}

func clampDetailBody(body []string) []string {
	if len(body) <= maxDetailBodyLines {
		return body
	}
	out := append([]string(nil), body[:maxDetailBodyLines]...)
	out = append(out, styleMuted.Render(strings.Repeat(" ", cellPad)+
		fmt.Sprintf("… truncated (%d more logical lines)", len(body)-maxDetailBodyLines)))
	return out
}

// detailVisualWindow returns painted rows for the viewport.
// Wrap-off paints only the visible slice; wrap-on expands once and caches.
func (m Model) detailVisualWindow(width, height, offset int) (rows []string, total int) {
	body := m.ensureDetailLogical(width)
	if m.detailWrap {
		visual := m.ensureDetailVisual(width, body)
		total = len(visual)
		if offset > max(0, total-1) {
			offset = max(0, total-1)
		}
		end := min(total, offset+height)
		if offset < total {
			rows = append([]string(nil), visual[offset:end]...)
		}
		return rows, total
	}
	total = len(body)
	if offset > max(0, total-1) {
		offset = max(0, total-1)
	}
	start := max(0, offset-detailViewOverscan)
	end := min(total, offset+height+detailViewOverscan)
	painted := make([]string, 0, end-start)
	for _, line := range body[start:end] {
		painted = append(painted, renderDetailRows(line, width, false)...)
	}
	rel := offset - start
	visEnd := min(len(painted), rel+height)
	if rel < len(painted) {
		rows = painted[rel:visEnd]
	}
	return rows, total
}

func (m Model) ensureDetailVisual(width int, body []string) []string {
	if m.detailCache == nil {
		m.detailCache = &detailRenderCache{}
	}
	if m.detailCache.visual != nil && m.detailCache.key == m.detailRenderKey(width) {
		return m.detailCache.visual
	}
	visual := expandDetailRows(body, width, true)
	m.detailCache.visual = visual
	return visual
}
