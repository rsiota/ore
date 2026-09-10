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

**Keys:** `?` help · `/` filter · `Enter` open · `b` blame · `f` follow · `esc` back · `Tab` detail · `q` quit

## Status

Usability slice: registry-driven help, grid filter, light chrome. Archaeology loop through blame is in place — see [ROADMAP.md](ROADMAP.md).

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
