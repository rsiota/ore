package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

// exCmdSpec is one ":" command. exCommands() is the source of truth for
// dispatch and the help Commands section.
type exCmdSpec struct {
	verbs []string // canonical first, then aliases
	desc  string
	usage string
	run   func(m *Model, args []string) tea.Cmd
}

func exCommands() []exCmdSpec {
	return []exCmdSpec{
		{
			verbs: []string{"blame"},
			desc:  "open blame grid for a path",
			usage: ":blame <path> [rev]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exBlame(args)
			},
		},
		{
			verbs: []string{"history", "hist"},
			desc:  "open path history (--follow)",
			usage: ":history <path>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exHistory(args)
			},
		},
		{
			verbs: []string{"couple", "cochange", "cocurrent"},
			desc:  "list commits where two paths co-occur (Often-with drill)",
			usage: ":couple <seed> <partner>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exCouple(args)
			},
		},
		{
			verbs: []string{"nohl", "nohls", "hlclear", "clearhl"},
			desc:  "turn off pickaxe match highlighting (keeps results)",
			usage: ":nohl",
			run: func(m *Model, _ []string) tea.Cmd {
				m.clearPickaxeHighlight()
				return nil
			},
		},
		{
			verbs: []string{"pickaxe", "pick", "S"},
			desc:  "search history for commits that added/removed a string (-S)",
			usage: ":pickaxe <string>  |  :S <string>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exPickaxe(args, git.PickaxeString)
			},
		},
		{
			verbs: []string{"G", "regexp", "regex"},
			desc:  "search history for commits matching a regexp pickaxe (-G)",
			usage: ":G <regexp>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exPickaxe(args, git.PickaxeRegexp)
			},
		},
		{
			verbs: []string{"evolve", "evolution", "evo"},
			desc:  "open line evolution stack for the selected blame line",
			usage: ":evolve",
			run: func(m *Model, _ []string) tea.Cmd {
				return m.exEvolve()
			},
		},
		{
			verbs: []string{"goto", "go"},
			desc:  "jump to a commit by hash prefix",
			usage: ":goto <hash>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exGoto(args)
			},
		},
		{
			verbs: []string{"branch", "br"},
			desc:  "view another branch tip (read-only; no checkout)",
			usage: ":branch [name]",
			run: func(m *Model, args []string) tea.Cmd {
				if len(args) == 0 {
					return m.openBranchPicker()
				}
				return m.switchViewRev(args[0])
			},
		},
		{
			verbs: []string{"theme"},
			desc:  "switch colour theme (light or dark)",
			usage: ":theme [light|dark]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exTheme(args)
			},
		},
		{
			verbs: []string{"set"},
			desc:  "change a setting (transparent_background)",
			usage: ":set transparent_background [on|off]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exSet(args)
			},
		},
		{
			verbs: []string{"refresh", "reload"},
			desc:  "reload the commit log and current view",
			usage: ":refresh",
			run: func(m *Model, _ []string) tea.Cmd {
				return m.refresh()
			},
		},
		{
			verbs: []string{"bookmark", "bm"},
			desc:  "bookmark the current view (optional name)",
			usage: ":bookmark [name]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exBookmark(args)
			},
		},
		{
			verbs: []string{"bookmarks", "marks"},
			desc:  "toggle the bookmarks panel (or clear)",
			usage: ":bookmarks [clear]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exBookmarks(args)
			},
		},
		{
			verbs: []string{"session"},
			desc:  "save, clear, or show the restored workspace session",
			usage: ":session [save|clear]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exSession(args)
			},
		},
		{
			verbs: []string{"help", "h"},
			desc:  "open the help overlay",
			usage: ":help",
			run: func(m *Model, _ []string) tea.Cmd {
				m.help.Show()
				m.status = "help"
				return nil
			},
		},
		{
			verbs: []string{"quit", "q"},
			desc:  "quit ore",
			usage: ":q",
			run: func(m *Model, _ []string) tea.Cmd {
				return m.beginQuit()
			},
		},
	}
}

func exLookup(verb string) *exCmdSpec {
	v := strings.ToLower(verb)
	cmds := exCommands()
	for i := range cmds {
		for _, alias := range cmds[i].verbs {
			if strings.ToLower(alias) == v {
				return &cmds[i]
			}
		}
	}
	return nil
}
