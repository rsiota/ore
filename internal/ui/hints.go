package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// hintFlashDuration is how long a pressed hint key stays cell-fg+bold.
const hintFlashDuration = 300 * time.Millisecond

// hintDescDuration is how long a pressed key's description is shown inline on
// the status bar after the key is pressed (a touch longer than the key flash,
// so the eye can land on the description).
const hintDescDuration = 1500 * time.Millisecond

// matchHint returns the individual key within a hint group that matches the
// pressed key, or "" if none. Each hint group is split on "/" to extract
// individual keys (e.g. "h/j/k/l" yields h, j, k, l). Multi-word keys like
// "ctrl+s" and "enter" are matched as-is. A literal space key is normalized
// to "space" to match the display hint.
func matchHint(hints []string, key string) string {
	if key == " " {
		key = "space"
	}
	for _, h := range hints {
		if h == key {
			return h
		}
		for _, k := range strings.Split(h, "/") {
			if k == key {
				return k
			}
		}
	}
	return ""
}

// hintSectionTitles returns registry section titles to search for a pressed
// key's description, ordered most-specific first.
func (m Model) hintSectionTitles() []string {
	if m.explorer.Opened() {
		return []string{"Relationships (g r)", "Global"}
	}
	switch m.main {
	case MainCommits:
		return []string{"Commit grid", "Navigation", "Global"}
	case MainFiles:
		return []string{"Files grid", "Navigation", "Global"}
	case MainHistory:
		return []string{"History grid", "Navigation", "Global"}
	case MainBlame:
		return []string{"Blame", "Navigation", "Global"}
	case MainLineEvo:
		return []string{"Line evolution", "Navigation", "Global"}
	default:
		return []string{"Navigation", "Global"}
	}
}

// hintDescription returns the registry description for a matched hint key in
// the current context, or "" if none. matchedKey is the individual key string
// returned by matchHint (e.g. "j", "ctrl+p"). It is matched against binding
// Tokens rather than Hint strings: Hint "/"-splits are ambiguous.
func (m Model) hintDescription(matchedKey string) string {
	for _, title := range m.hintSectionTitles() {
		for _, sec := range registry() {
			if sec.Title != title {
				continue
			}
			for _, b := range sec.Items {
				for _, t := range b.Tokens {
					if t == matchedKey {
						return b.Desc
					}
				}
			}
		}
	}
	return ""
}

// renderHintStrip styles status-bar hint groups, flashing the pressed key as
// cell fg+bold while idle keys stay muted (creel-style).
func renderHintStrip(hints []string, flashKey string, flashActive bool) string {
	keyStyle := styleMuted
	flashStyle := styleHintFlash
	sepStyle := styleMuted
	var b strings.Builder
	for i, group := range hints {
		if i > 0 {
			b.WriteString(sepStyle.Render("/"))
		}
		if group == "/" {
			if flashActive && group == flashKey {
				b.WriteString(flashStyle.Render(group))
			} else {
				b.WriteString(keyStyle.Render(group))
			}
			continue
		}
		for ki, k := range strings.Split(group, "/") {
			if ki > 0 {
				b.WriteString(sepStyle.Render("/"))
			}
			if flashActive && k == flashKey {
				b.WriteString(flashStyle.Render(k))
			} else {
				b.WriteString(keyStyle.Render(k))
			}
		}
	}
	return b.String()
}

// hintFlashActive reports whether the pressed-key flash should still paint.
func (m Model) hintFlashActive() bool {
	return m.hintFlash != "" && time.Since(m.hintFlashAt) < hintFlashDuration
}

// hintDescActive reports whether the inline description should still paint.
func (m Model) hintDescActive() bool {
	return m.hintDesc != "" && time.Since(m.hintDescAt) < hintDescDuration
}

// stageHintFlash records a pressed key for status-bar flash + description.
func (m *Model) stageHintFlash(key string) {
	hints := statusHintList(m.main, m.explorer.Opened())
	if matched := matchHint(hints, key); matched != "" {
		m.hintFlash = matched
		m.hintFlashAt = time.Now()
		if d := m.hintDescription(matched); d != "" {
			m.hintDesc = d
			m.hintDescAt = time.Now()
		} else {
			m.hintDesc = ""
		}
		return
	}
	m.hintFlash = ""
	m.hintDesc = ""
}

// truncateHintsStyled caps a styled hint strip to max visual columns.
func truncateHintsStyled(s string, max int) string {
	if max < 1 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(max).Render(s)
}
