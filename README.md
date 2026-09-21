# ore

A keyboard-first **Git archaeology** TUI — explore how code got here, not how to stage the next commit.

Sibling spirit to [creel](https://github.com/rsiota/creel): one fast Go binary, vim-native motions, results-grid thinking, relationship navigation. Read-only by design (for now).

```sh
go install github.com/rsiota/ore/cmd/ore@latest   # once published
# from a clone:
go run ./cmd/ore                                 # current repo
go run ./cmd/ore /path/to/repo
go run ./cmd/ore -C /path/to/repo
```

**Keys:** `ctrl+p` palette · `m` bookmark · `g m` / `ctrl+g` bookmarks · `D` zen/unified diff · `w` wrap · `[`/`]` zen context · `:` commands · `g r` relations · `?` help · `/` filter · `Enter` open · `b` blame · `f` follow · `F` evolve · `esc` back · `q` quit

**Commit grid:** soft graph · `h`/`l` columns · `o` cycle sort · `/` filters the active column

## Status

Commit grid is columnar (sort + column-scoped `/`). Ex commands (`:blame`, `:history`, `:goto`, …). See [ROADMAP.md](ROADMAP.md).

### Ex commands

| Command | Action |
|---------|--------|
| `:blame <path> [rev]` | Open blame (rev defaults to selection / HEAD) |
| `:couple <seed> <partner>` | Commits where paths co-occur (Often-with drill) |
| `:pickaxe <text>` / `:S <text>` | Commits that added/removed a string (`git log -S`); matches wash in detail/blame |
| `:G <regexp>` | Commits matching a regexp pickaxe (`git log -G`) |
| `:nohl` | Clear pickaxe match highlighting (keeps the results grid) |
| `:evolve` | Line evolution stack for the selected blame line (`F`) |
| `:history <path>` | Path history with rename follow + “was / moved from” edges |
| `:goto <hash>` | Jump to commit (prefix match) |
| `:bookmark [name]` | Bookmark the current view |
| `:bookmarks [clear]` | Toggle bookmarks panel (or clear all) |
| `:theme [light|dark]` | Switch colour theme (persisted) |
| `:set transparent_background [on|off]` | Leave terminal background unpainted |
| `:session [save|clear]` | Show, save, or clear restored workspace |
| `:help` | Open help |
| `:q` | Quit |

## Stack

- Go 1.26+
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- System `git` CLI for log / show / (later) blame and `--follow`

## Layout (target)

```
┌─ Sidebar ──────────┬─ Main Grid / View ──────────────────────────────┬─ Detail ─────┐
│  Branches / Tags   │  Commit list  or  File history  or  Blame grid   │  Diff /      │
│  or File tree      │  (tabular, sortable, filterable)                  │  Patch /     │
│  or Search results │                                                  │  Commit msg  │
│                    ├──────────────────────────────────────────────────┤              │
│                    │  Optional: Timeline strip or Hunk list           │              │
└────────────────────┴──────────────────────────────────────────────────┴──────────────┘
```

Wave 0 ships the main grid + detail only.

## License

MIT
