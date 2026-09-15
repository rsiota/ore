package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// defaultBgResetSeq is the SGR "default background" code. Injected in
// transparent_background mode so cells that previously held the theme bg
// fall back to the terminal default (respecting profile transparency).
const defaultBgResetSeq = "\x1b[49m"

// paintBackground fills every cell in view that has no explicit background
// colour with bg. Cells that set their own background (selection, stripes,
// washes, …) keep it. Adapted from creel so light themes stay readable on
// dark terminal profiles.
func paintBackground(view string, bg lipgloss.Color) string {
	seq := ansiBgSeq(bg)
	if seq == "" {
		return view
	}
	return paintBackgroundSeq(view, seq)
}

func paintBackgroundSeq(view string, seq string) string {
	if seq == "" {
		return view
	}

	var out strings.Builder
	out.Grow(len(view) + 64)

	explicit, injected := false, false
	pos := 0
	for {
		match := sgrRE.FindStringSubmatchIndex(view[pos:])
		if match == nil {
			writeLiteral(&out, view[pos:], seq, &explicit, &injected)
			break
		}
		start, end := match[0], match[1]
		if start > 0 {
			writeLiteral(&out, view[pos:pos+start], seq, &explicit, &injected)
		}
		out.WriteString(view[pos+start : pos+end])
		applySGR(view[pos+match[2]:pos+match[3]], &explicit, &injected)
		pos += end
	}
	return out.String()
}

func writeLiteral(out *strings.Builder, s, seq string, explicit, injected *bool) {
	for _, r := range s {
		if r == '\n' {
			*injected = false
			out.WriteRune(r)
			continue
		}
		if r >= 0x20 && !*explicit && !*injected {
			out.WriteString(seq)
			*injected = true
		}
		out.WriteRune(r)
	}
}

func applySGR(paramsStr string, explicit, injected *bool) {
	if paramsStr == "" {
		*explicit = false
		*injected = false
		return
	}
	raw := strings.Split(paramsStr, ";")
	params := make([]int, len(raw))
	for i, p := range raw {
		v, err := strconv.Atoi(p)
		if err != nil {
			return
		}
		params[i] = v
	}
	for i := 0; i < len(params); i++ {
		p := params[i]
		switch {
		case p == 0, p == 49:
			*explicit = false
			*injected = false
		case p == 48:
			*explicit = true
			*injected = false
			i += consumeColorSpec(params[i+1:])
		case p == 38:
			i += consumeColorSpec(params[i+1:])
		case (p >= 40 && p <= 47) || (p >= 100 && p <= 107):
			*explicit = true
			*injected = false
		}
	}
}

func consumeColorSpec(rest []int) int {
	if len(rest) == 0 {
		return 0
	}
	switch rest[0] {
	case 5:
		return 2
	case 2:
		return 4
	default:
		return 1
	}
}

var sgrRE = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func ansiBgSeq(c lipgloss.Color) string {
	if s := string(c); strings.HasPrefix(s, "#") {
		if r, g, b, ok := parseHexRGB(s); ok {
			return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
		}
	}
	if string(c) == "" {
		return ""
	}
	styled := lipgloss.NewStyle().Background(c).Render("|")
	idx := strings.IndexByte(styled, '|')
	if idx <= 0 {
		return ""
	}
	return styled[:idx]
}

func parseHexRGB(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(n>>16) & 0xff, int(n>>8) & 0xff, int(n) & 0xff, true
}

func (m Model) paintBg(view string) string {
	if m.transparentBg {
		return paintBackgroundSeq(view, defaultBgResetSeq)
	}
	return paintBackground(view, colorBg)
}
