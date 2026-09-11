package ui

// Binding is one documented keybinding (help + future palette).
type Binding struct {
	Display string   // what the user sees, e.g. "j/k"
	Tokens  []string // tea.KeyMsg.String() values the dispatch handles
	Desc    string
	Hint    string // compact status hint; empty = omit from hint line
}

// Section groups bindings under a heading.
type Section struct {
	Title string
	Items []Binding
}

// registry is the source of truth for the ? help overlay.
func registry() []Section {
	return []Section{
		{
			Title: "Global",
			Items: []Binding{
				{"?", []string{"?"}, "toggle help", "?"},
				{":", []string{":"}, "ex command line", ":"},
				{"ctrl+r", []string{"ctrl+r"}, "refresh commit log", ""},
				{"q / ctrl+c", []string{"q", "ctrl+c"}, "quit", "q"},
				{"tab", []string{"tab"}, "focus main ↔ detail", "tab"},
				{"esc / backspace", []string{"esc", "backspace"}, "go back / close overlay", "esc"},
				{"/", []string{"/"}, "filter current grid", "/"},
			},
		},
		{
			Title: "Navigation",
			Items: []Binding{
				{"j/k · ↑/↓", []string{"j", "k", "up", "down"}, "move row", "j/k"},
				{"g g / G", []string{"g", "G"}, "top / bottom", "gg/G"},
				{"g r", []string{"g", "r"}, "relationship explorer", "g r"},
				{"ctrl+d / ctrl+u", []string{"ctrl+d", "ctrl+u"}, "page down / up", ""},
				{"enter", []string{"enter"}, "open (commit→files→history; history→blame)", "enter"},
				{"l", []string{"l"}, "open (files / history / blame; not commits)", ""},
				{"b", []string{"b"}, "blame file at revision", "b"},
				{"f / g f", []string{"f", "g"}, "follow blame line backward", "f"},
			},
		},
		{
			Title: "Commit grid",
			Items: []Binding{
				{"h/l · ←/→", []string{"h", "l", "left", "right"}, "previous / next column", "h/l"},
				{"0 / $", []string{"0", "$"}, "first / last column", ""},
				{"o", []string{"o"}, "cycle sort on current column (asc→desc→off)", "o"},
				{"/", []string{"/"}, "filter current column", "/"},
				{"enter", []string{"enter"}, "open files for commit", "enter"},
			},
		},
		{
			Title: "Relationships (g r)",
			Items: []Binding{
				{"g r", []string{"g", "r"}, "open explorer for commit or blame line", "g r"},
				{"j/k", []string{"j", "k"}, "move in explorer", "j/k"},
				{"enter / l", []string{"enter", "l"}, "jump to commit or file history", "enter"},
				{"esc / h", []string{"esc", "h"}, "close explorer", "esc"},
				{"tab", []string{"tab"}, "focus main ↔ explorer", "tab"},
			},
		},
		{
			Title: "Views",
			Items: []Binding{
				{"commits", nil, "repository log (start)", ""},
				{"files", nil, "paths changed in the selected commit", ""},
				{"history", nil, "git log --follow for a path", ""},
				{"blame", nil, "line archaeology grid", ""},
			},
		},
		{
			Title: "Filter",
			Items: []Binding{
				{"/", []string{"/"}, "start filter (commits: active column)", "/"},
				{"enter", []string{"enter"}, "keep filter, leave input", ""},
				{"esc", []string{"esc"}, "clear filter", ""},
				{"backspace", []string{"backspace"}, "delete character", ""},
			},
		},
	}
}

// statusHints returns a short hint string for the current context.
func statusHints(main MainView, explorerOpen bool) string {
	if explorerOpen {
		return "enter open · esc close · tab focus · ?/help"
	}
	parts := []string{":cmd", "g r relations", "?/help", "/ filter", "esc back", "q quit"}
	switch main {
	case MainCommits:
		parts = append([]string{"h/l cols", "o sort", "enter files"}, parts...)
	case MainFiles:
		parts = append([]string{"enter history", "b blame"}, parts...)
	case MainHistory:
		parts = append([]string{"enter/b blame"}, parts...)
	case MainBlame:
		parts = append([]string{"f follow"}, parts...)
	}
	return stringsJoinHints(parts)
}

func stringsJoinHints(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}
