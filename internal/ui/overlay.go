package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// popupDim returns the fixed outer size for shared form-style popups
// (border included). Prefer palettePopupDim for the Ctrl+P overlay.
func popupDim() (w, h int) {
	const (
		formTabBarLines     = 1
		popupStandardFields = 6
		linesPerField       = 4
	)
	contentH := formTabBarLines + popupStandardFields*linesPerField + 1 // 26
	return 71, contentH + 2                                             // 28
}

// palettePopupDim is the outer size of the Ctrl+P command palette.
// Kept shorter than popupDim so the prompt plus maxPaletteItems rows fill
// the frame without empty padding at the bottom (matches creel).
func palettePopupDim() (w, h int) {
	return 71, maxPaletteItems + 3 // items + prompt + border
}

// placeOverlay overlays fg on top of bg at (x, y). ANSI-aware via ansi.Cut.
func placeOverlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	for i, fgLine := range fgLines {
		targetY := y + i
		if targetY < 0 || targetY >= len(bgLines) {
			continue
		}
		bgLines[targetY] = overlayLine(bgLines[targetY], fgLine, x)
	}
	return strings.Join(bgLines, "\n")
}

func overlayLine(bgLine, fgLine string, x int) string {
	bgWidth := lipgloss.Width(bgLine)
	fgWidth := lipgloss.Width(fgLine)
	if x >= bgWidth {
		return bgLine + strings.Repeat(" ", x-bgWidth) + fgLine
	}
	left := ansi.Cut(bgLine, 0, x)
	rightStart := x + fgWidth
	right := ""
	if rightStart < bgWidth {
		right = ansi.Cut(bgLine, rightStart, bgWidth)
	}
	return left + fgLine + right
}
