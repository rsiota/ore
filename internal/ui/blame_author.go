package ui

import (
	"hash/fnv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Quiet author hues for the blame author column only — low chroma, no reds or
// greens so they never compete with add/del washes. Light = darker text;
// dark = softer lights on dim chrome.
var (
	authorHuesLight = []lipgloss.Color{
		lipgloss.Color("#6f42c1"), // purple
		lipgloss.Color("#0550ae"), // blue
		lipgloss.Color("#8250df"), // violet
		lipgloss.Color("#bf3989"), // pink
		lipgloss.Color("#9a6700"), // amber
		lipgloss.Color("#0a7b83"), // teal
		lipgloss.Color("#8a4600"), // brown
		lipgloss.Color("#3b6ea8"), // steel
	}
	authorHuesDark = []lipgloss.Color{
		lipgloss.Color("#d2a8ff"), // purple
		lipgloss.Color("#79c0ff"), // blue
		lipgloss.Color("#e3b341"), // amber
		lipgloss.Color("#f778ba"), // pink
		lipgloss.Color("#39c5cf"), // teal
		lipgloss.Color("#a5d6ff"), // ice
		lipgloss.Color("#d2a8ff"), // lilac
		lipgloss.Color("#ffa657"), // peach
	}
)

func authorHueIndex(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(name)))
	n := len(authorHuesLight)
	if activeThemeName == "dark" {
		n = len(authorHuesDark)
	}
	if n == 0 {
		return 0
	}
	return int(h.Sum32() % uint32(n))
}

func authorColor(name string) lipgloss.Color {
	i := authorHueIndex(name)
	if activeThemeName == "dark" {
		return authorHuesDark[i]
	}
	return authorHuesLight[i]
}

// authorNameStyle is foreground-only so age washes and diff chrome stay primary.
func authorNameStyle(name string) lipgloss.Style {
	if strings.TrimSpace(name) == "" {
		return styleMuted
	}
	return lipgloss.NewStyle().Foreground(authorColor(name))
}
