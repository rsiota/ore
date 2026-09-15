package ui

import "github.com/charmbracelet/lipgloss"

const defaultThemeName = "light"

// lightPalette is the shipped GitHub-light chrome — neutral slate selection so
// semantic colours (diff, age, hash) stand out.
var lightPalette = colorPalette{
	primary:         lipgloss.Color("#24292f"),
	fg:              lipgloss.Color("#1f2328"),
	muted:           lipgloss.Color("#656d76"),
	label:           lipgloss.Color("#57606a"),
	border:          lipgloss.Color("#d0d7de"),
	borderFocused:   lipgloss.Color("#6e7781"),
	borderUnfocused: lipgloss.Color("#afb8c1"),
	bg:              lipgloss.Color("#ffffff"),
	cursorRow:       lipgloss.Color("#e4e5e5"),
	stripe:          lipgloss.Color("#f2f2f2"),
	err:             lipgloss.Color("#cf222e"),
	hash:            lipgloss.Color("#0550ae"),
	graphLine:       lipgloss.Color("#6e7781"),
	graphNode:       lipgloss.Color("#afb8c1"),
	add:             lipgloss.Color("#1a7f37"),
	del:             lipgloss.Color("#cf222e"),
	addWash:         lipgloss.Color("#dafbe1"),
	delWash:         lipgloss.Color("#ffebe9"),
	zenHunk:         lipgloss.Color("#afb8c1"),
	filterBg:        lipgloss.Color("#fff8c5"),
	ageNew:          lipgloss.Color("#eef6f0"),
	ageMid:          lipgloss.Color("#f6f1e7"),
	ageOld:          lipgloss.Color("#f0e6e4"),
}

// darkPalette is a GitHub-dark counterpart: same neutral selection stance,
// quiet age/diff washes tuned for a dim terminal.
var darkPalette = colorPalette{
	primary:         lipgloss.Color("#c9d1d9"),
	// Soft neutral body text — less cool/bright than GitHub’s #e6edf3.
	fg:              lipgloss.Color("#c4cbd4"),
	muted:           lipgloss.Color("#8b949e"),
	label:           lipgloss.Color("#7d8590"),
	border:          lipgloss.Color("#30363d"),
	borderFocused:   lipgloss.Color("#8b949e"),
	borderUnfocused: lipgloss.Color("#484f58"),
	bg:              lipgloss.Color("#0d1117"),
	cursorRow:       lipgloss.Color("#21262d"),
	stripe:          lipgloss.Color("#161b22"),
	err:             lipgloss.Color("#f85149"),
	hash:            lipgloss.Color("#58a6ff"),
	// Graph a step brighter than muted chrome so lanes stay visible on bg.
	graphLine:       lipgloss.Color("#8b949e"),
	graphNode:       lipgloss.Color("#b1bac4"),
	add:             lipgloss.Color("#3fb950"),
	del:             lipgloss.Color("#f85149"),
	addWash:         lipgloss.Color("#12261e"),
	delWash:         lipgloss.Color("#2d1214"),
	zenHunk:         lipgloss.Color("#484f58"),
	filterBg:        lipgloss.Color("#3d2e00"),
	ageNew:          lipgloss.Color("#12261e"),
	ageMid:          lipgloss.Color("#2a2118"),
	ageOld:          lipgloss.Color("#2a1819"),
}

var themes = map[string]colorPalette{
	"light": lightPalette,
	"dark":  darkPalette,
}

var activeThemeName = defaultThemeName

// themeNames returns known themes in display order (default first).
func themeNames() []string {
	return []string{"light", "dark"}
}

func resolveThemeName(name string) (string, bool) {
	for _, t := range themeNames() {
		if equalFoldASCII(t, name) {
			return t, true
		}
	}
	return "", false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func paletteForTheme(name string) colorPalette {
	if p, ok := themes[name]; ok {
		return p
	}
	return lightPalette
}

// applyTheme sets package styles from a named palette. Unknown / empty names
// fall back to light so a bad config never blocks startup.
func applyTheme(name string) {
	resolved := defaultThemeName
	if t, ok := resolveThemeName(name); ok {
		resolved = t
	}
	activeThemeName = resolved
	applyPalette(paletteForTheme(resolved))
}
