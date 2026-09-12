package ui

import "github.com/charmbracelet/lipgloss"

// Light / GitHub-light chrome — readable on pale terminal themes.
// Selection chrome is neutral slate so semantic colours (diff, age, hash) stand out.
const borderOverhead = 2

func panelBorder() lipgloss.Border {
	return lipgloss.NormalBorder()
}

var (
	colorPrimary         = lipgloss.Color("#24292f") // slate fill (cursor cell, tabs)
	colorBorderFocused   = lipgloss.Color("#6e7781") // focused pane frame
	colorBorderUnfocused = lipgloss.Color("#afb8c1") // unfocused pane frame
	colorBorder          = lipgloss.Color("#d0d7de") // inner grid lines
	colorBg              = lipgloss.Color("#ffffff")
	colorRowFocusBg      = lipgloss.Color("#eaeef2")
	colorStripe          = lipgloss.Color("#fafbfc") // zebra row tint (subtle)

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76"))
	// List / soft focus wash (explorer selection) — neutral, not accent blue.
	styleFocus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1f2328")).
			Background(colorRowFocusBg)
	// Grid cursor cell — white text on slate.
	styleCursorCell = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary)
	// Pane / status tab when focused — slate pill, white text.
	styleSelected = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary).
			Padding(0, 1)
	styleHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76")).Bold(true)
	// Grid column headers — slate, bold; selected col underlines the word only.
	styleGridHeader = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	styleGridBorder = lipgloss.NewStyle().Foreground(colorBorder)
	styleStripe     = lipgloss.NewStyle().Background(colorStripe)
	styleErr        = lipgloss.NewStyle().Foreground(lipgloss.Color("#cf222e"))
	styleHash       = lipgloss.NewStyle().Foreground(lipgloss.Color("#0550ae"))

	// Single-colour graph: solid lines, dots a touch lighter (terminal “opacity”).
	graphLineColor = lipgloss.Color("#6e7781")
	graphNodeColor = lipgloss.Color("#afb8c1")

	styleAdd = lipgloss.NewStyle().Foreground(lipgloss.Color("#1a7f37"))
	styleDel = lipgloss.NewStyle().Foreground(lipgloss.Color("#cf222e"))

	styleAddWash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1a7f37")).
			Background(lipgloss.Color("#dafbe1"))
	styleDelWash = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cf222e")).
			Background(lipgloss.Color("#ffebe9"))

	// Detail pane hierarchy: meta muted, subject bold, hunk headers labelled.
	styleDiffMeta = lipgloss.NewStyle().Foreground(lipgloss.Color("#656d76"))
	styleDiffHunk = lipgloss.NewStyle().Foreground(lipgloss.Color("#57606a")).Bold(true)

	styleHelpTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(colorPrimary)
	styleHelpSection = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleFilter      = lipgloss.NewStyle().Foreground(lipgloss.Color("#1f2328")).Background(lipgloss.Color("#fff8c5"))
	// Soft row wash on focused blame / related rows.
	styleRowFocus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1f2328")).
			Background(colorRowFocusBg)
)
