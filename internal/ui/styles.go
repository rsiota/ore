package ui

import "github.com/charmbracelet/lipgloss"

// Selection chrome is neutral slate so semantic colours (diff, age, hash) stand out.
const borderOverhead = 2

func panelBorder() lipgloss.Border {
	return lipgloss.NormalBorder()
}

// colorPalette is every colour a theme provides. Package styles are rebuilt
// from it via applyPalette so :theme switches live without per-widget wiring.
type colorPalette struct {
	primary         lipgloss.Color
	fg              lipgloss.Color
	muted           lipgloss.Color
	label           lipgloss.Color
	border          lipgloss.Color
	borderFocused   lipgloss.Color
	borderUnfocused lipgloss.Color
	bg              lipgloss.Color
	cursorRow       lipgloss.Color
	stripe          lipgloss.Color
	err             lipgloss.Color
	hash            lipgloss.Color
	graphLine       lipgloss.Color
	graphNode       lipgloss.Color
	add             lipgloss.Color
	del             lipgloss.Color
	addWash         lipgloss.Color
	delWash         lipgloss.Color
	zenHunk         lipgloss.Color
	filterBg        lipgloss.Color
	ageNew          lipgloss.Color
	ageMid          lipgloss.Color
	ageOld          lipgloss.Color
}

// Colour slots — assigned by applyPalette (init + theme switch).
var (
	colorPrimary         lipgloss.Color
	colorFg              lipgloss.Color
	colorMuted           lipgloss.Color
	colorBorderFocused   lipgloss.Color
	colorBorderUnfocused lipgloss.Color
	colorBorder          lipgloss.Color
	colorBg              lipgloss.Color
	colorRowFocusBg      lipgloss.Color
	colorStripe          lipgloss.Color
	colorAgeNewBg        lipgloss.Color
	colorAgeMidBg        lipgloss.Color
	colorAgeOldBg        lipgloss.Color

	graphLineColor lipgloss.Color
	graphNodeColor lipgloss.Color

	styleTitle      lipgloss.Style
	styleMuted      lipgloss.Style
	styleCell       lipgloss.Style // default grid body — explicit fg so paintBg stays readable
	styleFocus      lipgloss.Style
	styleCursorCell lipgloss.Style
	styleSelected   lipgloss.Style
	styleHeader     lipgloss.Style
	styleGridHeader lipgloss.Style
	styleGridBorder lipgloss.Style
	styleStripe     lipgloss.Style
	styleErr        lipgloss.Style
	styleHash       lipgloss.Style
	styleAdd        lipgloss.Style
	styleDel        lipgloss.Style
	styleAddWash    lipgloss.Style
	styleDelWash    lipgloss.Style
	styleDiffMeta   lipgloss.Style
	styleDiffHunk   lipgloss.Style
	styleZenHunk    lipgloss.Style
	styleZenFile    lipgloss.Style
	styleHelpTitle  lipgloss.Style
	styleHelpSection lipgloss.Style
	styleFilter     lipgloss.Style
	styleRowFocus   lipgloss.Style
)

func init() {
	applyTheme(defaultThemeName)
}

func applyPalette(p colorPalette) {
	colorPrimary = p.primary
	colorFg = p.fg
	colorMuted = p.muted
	colorBorderFocused = p.borderFocused
	colorBorderUnfocused = p.borderUnfocused
	colorBorder = p.border
	colorBg = p.bg
	colorRowFocusBg = p.cursorRow
	colorStripe = p.stripe
	colorAgeNewBg = p.ageNew
	colorAgeMidBg = p.ageMid
	colorAgeOldBg = p.ageOld
	graphLineColor = p.graphLine
	graphNodeColor = p.graphNode

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleMuted = lipgloss.NewStyle().Foreground(p.muted)
	styleCell = lipgloss.NewStyle().Foreground(p.fg)
	styleFocus = lipgloss.NewStyle().
		Foreground(p.fg).
		Background(p.cursorRow)
	styleCursorCell = lipgloss.NewStyle().
		Foreground(p.bg).
		Background(p.primary)
	styleSelected = lipgloss.NewStyle().
		Foreground(p.bg).
		Background(p.primary).
		Padding(0, 1)
	styleHeader = lipgloss.NewStyle().Foreground(p.muted).Bold(true)
	styleGridHeader = lipgloss.NewStyle().Foreground(p.primary).Bold(true)
	styleGridBorder = lipgloss.NewStyle().Foreground(p.border)
	// Stripe must set fg too — background alone leaves terminal-default text,
	// which reads as dark-on-dark after paintBg on a light terminal profile.
	styleStripe = lipgloss.NewStyle().Foreground(p.fg).Background(p.stripe)
	styleErr = lipgloss.NewStyle().Foreground(p.err)
	styleHash = lipgloss.NewStyle().Foreground(p.hash)

	styleAdd = lipgloss.NewStyle().Foreground(p.add)
	styleDel = lipgloss.NewStyle().Foreground(p.del)
	styleAddWash = lipgloss.NewStyle().Foreground(p.add).Background(p.addWash)
	styleDelWash = lipgloss.NewStyle().Foreground(p.del).Background(p.delWash)

	styleDiffMeta = lipgloss.NewStyle().Foreground(p.muted)
	styleDiffHunk = lipgloss.NewStyle().Foreground(p.label).Bold(true)
	styleZenHunk = lipgloss.NewStyle().Foreground(p.zenHunk)
	styleZenFile = lipgloss.NewStyle().Foreground(p.primary)

	styleHelpTitle = lipgloss.NewStyle().Bold(true).Foreground(p.bg).Background(p.primary)
	styleHelpSection = lipgloss.NewStyle().Bold(true).Foreground(p.primary)
	styleFilter = lipgloss.NewStyle().Foreground(p.fg).Background(p.filterBg)
	styleRowFocus = lipgloss.NewStyle().
		Foreground(p.fg).
		Background(p.cursorRow)
}
