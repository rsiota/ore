package ui

import "strings"

// Binding is one documented keybinding (help + command palette).
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

// registry is the source of truth for the ? help overlay and Ctrl+P palette.
func registry() []Section {
	return []Section{
		{
			Title: "Global",
			Items: []Binding{
				{"?", []string{"?"}, "toggle help", "?"},
				{":", []string{":"}, "ex command line", ":"},
				{"ctrl+p", []string{"ctrl+p"}, "command palette", "ctrl+p"},
				{"ctrl+r", []string{"ctrl+r"}, "refresh commit log", ""},
				{"D", []string{"D"}, "cycle diff view (zen ↔ unified)", "D"},
				{"H", []string{"H"}, "toggle hunk list strip", "H"},
				{"[/]", []string{"[", "]"}, "zen context · or hunk prev/next when H on", ""},
				{"w", []string{"w"}, "toggle diff soft-wrap", "w"},
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
				{"g b", []string{"g", "b"}, "switch viewed branch (read-only)", "gb"},
				{"g m / ctrl+g", []string{"g", "m", "ctrl+g"}, "open bookmarks", "gm"},
				{"m", []string{"m"}, "bookmark current view", "m"},
				{"g r", []string{"g", "r"}, "relationship explorer", "gr"},
				{"p / c / u", []string{"p", "c", "u"}, "DAG parent / child / merge-base", "p/c/u"},
				{"g p / g c / g u", []string{"g", "p", "c", "u"}, "DAG parent / child / merge-base", ""},
				{"ctrl+d / ctrl+u", []string{"ctrl+d", "ctrl+u"}, "page down / up", ""},
				{"enter", []string{"enter"}, "open (commit→files→history; history→blame)", "enter"},
				{"l", []string{"l"}, "open (blame from non-grid views)", ""},
				{"b", []string{"b"}, "blame file at revision", "b"},
				{"f / g f", []string{"f", "g"}, "follow blame line backward one hop", "f"},
				{"F", []string{"F"}, "line evolution stack from blame", "F"},
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
			Title: "Files grid",
			Items: []Binding{
				{"h/l · ←/→", []string{"h", "l", "left", "right"}, "previous / next column", "h/l"},
				{"0 / $", []string{"0", "$"}, "first / last column", ""},
				{"o", []string{"o"}, "cycle sort on current column (asc→desc→off)", "o"},
				{"/", []string{"/"}, "filter current column", "/"},
				{"enter", []string{"enter"}, "open path history", "enter"},
				{"b", []string{"b"}, "blame file at revision", "b"},
			},
		},
		{
			Title: "History grid",
			Items: []Binding{
				{"h/l · ←/→", []string{"h", "l", "left", "right"}, "previous / next column", "h/l"},
				{"0 / $", []string{"0", "$"}, "first / last column", ""},
				{"o", []string{"o"}, "cycle sort on current column (asc→desc→off)", "o"},
				{"/", []string{"/"}, "filter current column", "/"},
				{"enter / b", []string{"enter", "b"}, "blame path at commit", "enter"},
			},
		},
		{
			Title: "Blame",
			Items: []Binding{
				{"h/l · ←/→", []string{"h", "l", "left", "right"}, "column; on code fold/unfold meta (default: line+commit+code)", "h/l"},
				{"0 / $", []string{"0", "$"}, "first / last column", ""},
				{"< / >", []string{"<", ">", ",", "."}, "scroll code left / right", "<>"},
				{"o", []string{"o"}, "cycle sort on current column (asc→desc→off)", "o"},
				{"/", []string{"/"}, "filter current column", "/"},
				{"f / g f", []string{"f", "g"}, "follow line backward one hop", "f"},
				{"F", []string{"F"}, "open line evolution stack", "F"},
			},
		},
		{
			Title: "Line evolution",
			Items: []Binding{
				{"F", []string{"F"}, "open from blame line", "F"},
				{"j/k", []string{"j", "k"}, "move in evolution stack", "j/k"},
				{"enter", []string{"enter"}, "open blame at this step", "enter"},
				{"esc", []string{"esc"}, "back to blame", "esc"},
			},
		},
		{
			Title: "Hunk list (H)",
			Items: []Binding{
				{"H", []string{"H"}, "toggle bottom hunk strip", "H"},
				{"tab", []string{"tab"}, "focus hunks (when strip open)", "tab"},
				{"j/k", []string{"j", "k"}, "next / previous hunk", "j/k"},
				{"[/]", []string{"[", "]"}, "previous / next hunk", "[/]"},
				{"enter", []string{"enter"}, "jump detail to hunk", "enter"},
				{"g / G", []string{"g", "G"}, "first / last hunk", ""},
				{"esc", []string{"esc"}, "return focus to main grid", "esc"},
			},
		},
		{
			Title: "Yank browser",
			Items: []Binding{
				{"tab", []string{"tab"}, "enter / leave yank (blame code → detail)", "tab"},
				{"hjkl", []string{"h", "j", "k", "l"}, "move by character / line", "hjkl"},
				{"w/b/e", []string{"w", "b", "e"}, "word motions", "w/b/e"},
				{"0 / $", []string{"0", "$"}, "line start / end", ""},
				{"g g / G", []string{"g", "G"}, "top / bottom", "gg/G"},
				{"f/t/F/T", []string{"f", "t", "F", "T"}, "find char on line", ""},
				{"/", []string{"/"}, "search in yank surface", "/"},
				{"n / N", []string{"n", "N"}, "next / previous search match", ""},
				{"v / V", []string{"v", "V"}, "visual char / line", "v/V"},
				{"y / Y", []string{"y", "Y"}, "yank selection / line (yy yw y$)", "y"},
				{"[/]", []string{"[", "]"}, "previous / next hunk (detail)", "[/]"},
				{"{/}", []string{"{", "}"}, "previous / next hunk (detail)", ""},
				{"esc", []string{"esc"}, "leave visual, then leave yank", "esc"},
			},
		},
		{
			Title: "Relationships (g r)",
			Items: []Binding{
				{"g r", []string{"g", "r"}, "open explorer for commit or blame line", "gr"},
				{"j/k", []string{"j", "k"}, "move in explorer", "j/k"},
				{"l", []string{"l"}, "expand commit / dive into nested", "l"},
				{"h", []string{"h"}, "collapse nested / close explorer", "h"},
				{"enter", []string{"enter"}, "jump to commit, file history, or Often-with couple", "enter"},
				{"esc", []string{"esc"}, "close explorer", "esc"},
				{"tab", []string{"tab"}, "focus main ↔ explorer", "tab"},
			},
		},
		{
			Title: "Pickaxe",
			Items: []Binding{
				{":pickaxe / :S", []string{":"}, "commits that added/removed a string", ""},
				{":G", []string{":"}, "commits matching a regexp pickaxe", ""},
				{":couple", []string{":"}, "commits where two paths co-occur", ""},
				{":nohl", []string{":"}, "clear pickaxe match washes", ""},
				{"j/k", []string{"j", "k"}, "move in pickaxe/couple results", "j/k"},
				{"enter", []string{"enter"}, "open path history (or files)", "enter"},
				{"b", []string{"b"}, "blame hit path at commit", "b"},
				{"esc", []string{"esc"}, "leave results", "esc"},
			},
		},
		{
			Title: "Views",
			Items: []Binding{
				{"commits", nil, "repository log (start)", ""},
				{"files", nil, "paths changed in the selected commit", ""},
				{"history", nil, "git log --follow for a path", ""},
				{"blame", nil, "line archaeology grid", ""},
				{"evolve", nil, "line provenance stack from blame (F)", ""},
				{"pickaxe", nil, "content history search (:pickaxe / :G)", ""},
				{"couple", nil, "co-change intersecting commits (Often-with / :couple)", ""},
			},
		},
		{
			Title: "Filter",
			Items: []Binding{
				{"/", []string{"/"}, "start filter (commits: active column)", "/"},
				{"enter", []string{"enter"}, "keep filter, leave input", "enter"},
				{"esc", []string{"esc"}, "clear filter", "esc"},
				{"backspace", []string{"backspace"}, "delete character", ""},
			},
		},
	}
}

// hintsForSection collects non-empty Hint strings from a registry section.
func hintsForSection(title string) []string {
	for _, sec := range registry() {
		if sec.Title != title {
			continue
		}
		var hints []string
		for _, b := range sec.Items {
			if b.Hint != "" {
				hints = append(hints, b.Hint)
			}
		}
		return hints
	}
	return nil
}

// statusHintList returns compact, key-only hint groups for the status bar
// (creel-style: j/k, enter, gr, …).
func statusHintList(main MainView, explorerOpen bool) []string {
	if explorerOpen {
		return hintsForSection("Relationships (g r)")
	}
	switch main {
	case MainCommits:
		return append([]string{"j/k"}, append(hintsForSection("Commit grid"), "gb", "gr", "D", "w", "tab", "ctrl+p", "?", "esc", "q")...)
	case MainFiles:
		return append([]string{"j/k"}, append(hintsForSection("Files grid"), "gb", "gr", "D", "w", "tab", "ctrl+p", "?", "esc", "q")...)
	case MainHistory:
		return append([]string{"j/k"}, append(hintsForSection("History grid"), "gb", "D", "w", "tab", "ctrl+p", "?", "esc", "q")...)
	case MainBlame:
		return append([]string{"j/k"}, append(hintsForSection("Blame"), "gb", "gr", "D", "w", "tab", "ctrl+p", "?", "esc", "q")...)
	case MainLineEvo:
		return append([]string{"j/k"}, append(hintsForSection("Line evolution"), "D", "w", "tab", "ctrl+p", "?", "esc", "q")...)
	case MainPickaxe:
		return append([]string{"j/k"}, append(hintsForSection("Pickaxe"), "D", "w", "tab", "ctrl+p", "?", "esc", "q")...)
	default:
		return []string{"j/k", "enter", "tab", "D", "w", "ctrl+p", "?", "esc", "q"}
	}
}

// statusHintListFocus is statusHintList with detail-yank focus awareness.
func statusHintListFocus(main MainView, explorerOpen bool, focus Focus) []string {
	if focus == FocusHunks {
		return hintsForSection("Hunk list (H)")
	}
	if focus == FocusDetail || focus == FocusBlameYank {
		return hintsForSection("Yank browser")
	}
	return statusHintList(main, explorerOpen)
}

// statusHints joins contextual key hints with "/" like creel's status bar.
func statusHints(main MainView, explorerOpen bool) string {
	return strings.Join(statusHintList(main, explorerOpen), "/")
}
