package ui

import "github.com/charmbracelet/lipgloss"

// Light / GitHub-light chrome — readable on pale terminal themes.
// Pane borders follow creel: primary blue when focused, faded grey otherwise.
const borderOverhead = 2

func panelBorder() lipgloss.Border {
	return lipgloss.NormalBorder()
}

var (
	colorPrimary         = lipgloss.Color("#0969da")
	colorBorderUnfocused = lipgloss.Color("#e1e4e8")
	colorBorder          = lipgloss.Color("#d0d7de") // inner grid lines (creel light border)
	colorBg              = lipgloss.Color("#ffffff")
	colorRowFocusBg      = lipgloss.Color("#eaeef2")
	colorStripe          = lipgloss.Color("#f5f7f9") // zebra row tint (creel light)

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76"))
	// List / soft focus wash (files, history, explorer selection).
	styleFocus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1f2328")).
			Background(lipgloss.Color("#ddf4ff"))
	// Grid cursor cell — creel results: white text on primary blue.
	styleCursorCell = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorPrimary)
	// Pane title when focused — creel selected-table chrome: blue pill, white text, word-scoped.
	styleSelected = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary).
			Padding(0, 1)
	styleHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76")).Bold(true)
	// Grid column headers — creel: primary blue, bold; selected col underlines the word only.
	styleGridHeader = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	styleGridBorder = lipgloss.NewStyle().Foreground(colorBorder)
	styleStripe     = lipgloss.NewStyle().Background(colorStripe)
	styleErr        = lipgloss.NewStyle().Foreground(lipgloss.Color("#cf222e"))
	styleHash       = lipgloss.NewStyle().Foreground(lipgloss.Color("#0550ae"))

	styleAdd = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a7f37"))
	styleDel = lipgloss.NewStyle().Foreground(lipgloss.Color("#cf222e"))

	styleAddWash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a7f37")).
			Background(lipgloss.Color("#dafbe1"))
	styleDelWash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cf222e")).
			Background(lipgloss.Color("#ffebe9"))

	styleHelpTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(colorPrimary)
	styleHelpSection = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleFilter      = lipgloss.NewStyle().Foreground(lipgloss.Color("#1f2328")).Background(lipgloss.Color("#fff8c5"))
	// Cursor row (non-active cell) — creel colorCursorRow wash on light themes.
	styleRowFocus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1f2328")).
			Background(colorRowFocusBg)
)
