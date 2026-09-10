# Architecture

## Tech stack

- **Go 1.26+**
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Git**: system `git` CLI via `os/exec` (correct blame / `--follow` / rename detection)
- **Config**: `~/.config/ore/` (Wave 0: directory only)

## Layout

```
cmd/ore/           entry point
internal/git/      repository access (CLI wrappers)
internal/ui/       Bubble Tea UI
internal/config/   config directory helpers
internal/version/  ldflags version string
```

## Design notes

- Domain boundary mirrors creel’s `internal/db`: UI talks to `*git.Repo`, never shells out itself.
- Prefer copying creel interaction patterns (registry, palette, session, themes) over a shared module until a second product proves the abstraction.
- Read-only product surface; do not grow write-path features in early waves.
- Navigation stack: **commits → files → history → blame** (`Enter`/`l` / `b` forward, `esc` back). History uses `git log --follow`; blame uses `git blame --porcelain`; detail uses `Show` / `ShowPath`. Line follow (`f` / `g f`) reloads blame at porcelain `previous`.
