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
internal/bookmarks/ per-repo named views
internal/version/  ldflags version string
```

## Design notes

- Domain boundary mirrors creel’s `internal/db`: UI talks to `*git.Repo`, never shells out itself.
- Prefer copying creel interaction patterns (registry, palette, session, themes) over a shared module until a second product proves the abstraction.
- Read-only product surface; do not grow write-path features in early waves.
- Navigation stack: **commits → files → history → blame → evolve** (`Enter` / `b` forward; `F` opens the line evolution stack from blame; `l` still opens on files/history/blame; `esc` back). Pickaxe (`:pickaxe` / `:S` / `:G`) opens a content-history results grid and washes matching spans in detail/blame until `:nohl` or leaving pickaxe. History uses `git log --follow --name-status` (`PathCommit` with path-at-rev and rename/copy edges); blame uses `git blame --porcelain`; evolution walks porcelain `previous`; detail uses `Show` / `ShowPath` with the path at the selected revision. One-hop follow remains `f` / `g f`. Rename edges surface as “was X” / “moved from Y” in the history path column, blame status, and line `g r`.
- Commit grid: shared `Grid` widget — cell cursor (`h`/`l`), `o` cycles sort on the focused column, `/` filters that column (frozen until cleared).
- Discoverability: `registry()` drives the `?` help overlay; `/` live-filters the current main grid. Status-bar hints flash the pressed key (cell fg+bold) and briefly show its registry description.
- Relationships: `g r` docks a right-pane explorer (parents/children/files, or blame-line neighbourhood). `l` lazily expands a commit inline; `h` collapses; `Enter` jumps to a commit or file history. Commit and blame-line views include an **Often with** section of co-changed file hot spots; `Enter` on a hot spot opens intersecting commits (`:couple`). File rows and line Previous/history show rename edges when known.
- Detail / blame yank: `Tab` focuses a readonly VIEW browser (creel-inspired chords, not the textarea engine): `hjkl`/`wbe`/`0$`/`gg`/`G`, `f/t`, pane-local `/`, visual `v`/`V`, `y`/`Y`/`yy`/`yw`/`y$` to the system clipboard; reverse block cursor. On blame, `Tab` enters the code column first, then detail; elsewhere `Tab` enters detail. `[`/`]` (and `{`/`}`) jump hunk headers in detail only; `esc` peels visual then returns focus to the main grid. On the main grid, `[`/`]` still adjust zen diff context.
- Ex commands: `:` opens a vim-style prompt; `exCommands()` drives dispatch and the help Commands list (`:blame`, `:history`, `:goto`, `:theme`, `:session`, …).
- Themes: `light` (default) and `dark` palettes via `applyTheme`; `:theme` switches live and writes `theme:` to config. View paints theme `bg` so light stays readable on dark terminals; `:set transparent_background on` skips that paint so the terminal’s own background / transparency shows through.
- Session restore: quit / `:q` writes `~/.config/ore/sessions/<repo>.json` (view tip, main pane, commit, path, blame line, diff chrome). Reopen restores asynchronously after the commit log loads; `:session clear` drops it.
- Bookmarks: per-repo named views in `~/.config/ore/bookmarks/` (creel-style fuzzy popup). `m` / `:bookmark [name]` saves the current session snapshot; `g m` / `ctrl+g` / `:bookmarks` opens the panel; Enter jumps via the same restore path as sessions; `d` deletes.
