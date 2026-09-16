# Architecture

## Tech stack

- **Go 1.26+**
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Git**: system `git` CLI via `os/exec` (correct blame / `--follow` / rename detection)
- **Config**: `~/.config/ore/config.yaml` (theme; more settings later)

## Layout

```
cmd/ore/           entry point
internal/git/      repository access (CLI wrappers)
internal/ui/       Bubble Tea UI
internal/config/   config directory helpers
internal/session/  per-repo workspace restore
internal/version/  ldflags version string
```

## Design notes

- Domain boundary mirrors creel’s `internal/db`: UI talks to `*git.Repo`, never shells out itself.
- Prefer copying creel interaction patterns (registry, palette, session, themes) over a shared module until a second product proves the abstraction.
- Read-only product surface; do not grow write-path features in early waves.
- Navigation stack: **commits → files → history → blame** (`Enter` / `b` forward; `l` still opens on files/history/blame; `esc` back). History uses `git log --follow`; blame uses `git blame --porcelain`; detail uses `Show` / `ShowPath`. Line follow (`f` / `g f`) reloads blame at porcelain `previous`.
- Commit grid: shared `Grid` widget — cell cursor (`h`/`l`), `o` cycles sort on the focused column, `/` filters that column (frozen until cleared).
- Discoverability: `registry()` drives the `?` help overlay; `/` live-filters the current main grid. Status-bar hints flash the pressed key (cell fg+bold) and briefly show its registry description.
- Relationships: `g r` docks a right-pane explorer (parents/children/files, or blame-line neighbourhood). `l` lazily expands a commit inline; `h` collapses; `Enter` jumps to a commit or file history.
- Ex commands: `:` opens a vim-style prompt; `exCommands()` drives dispatch and the help Commands list (`:blame`, `:history`, `:goto`, `:theme`, `:session`, …).
- Themes: `light` (default) and `dark` palettes via `applyTheme`; `:theme` switches live and writes `theme:` to config. View paints theme `bg` so light stays readable on dark terminals; `:set transparent_background on` skips that paint so the terminal’s own background / transparency shows through.
- Session restore: quit / `:q` writes `~/.config/ore/sessions/<repo>.json` (view tip, main pane, commit, path, blame line, diff chrome). Reopen restores asynchronously after the commit log loads; `:session clear` drops it.
