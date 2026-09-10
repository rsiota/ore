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

**Keys (Wave 2):** `Enter`/`l` open · `b` blame · `f`/`g f` follow line · `esc` back · `Tab` detail · `j`/`k` · `q`

## Status

Wave 2 started: blame grid with age washes and line follow. See [ROADMAP.md](ROADMAP.md).

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
