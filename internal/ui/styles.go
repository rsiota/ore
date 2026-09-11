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
	colorBg              = lipgloss.Color("#ffffff")

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76"))
	styleFocus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1f2328")).
			Background(lipgloss.Color("#ddf4ff"))
	// Pane title when focused — creel selected-table chrome: blue pill, white text, word-scoped.
	styleSelected = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary).
			Padding(0, 1)
	styleHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76")).Bold(true)
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("#cf222e"))
	styleHash   = lipgloss.NewStyle().Foreground(lipgloss.Color("#0550ae"))

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
	// Selected row, non-active cell — softer than styleFocus (active cell).
	styleRowFocus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1f2328")).
			Background(lipgloss.Color("#eaeef2"))
)
