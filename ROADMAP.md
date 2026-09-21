# ore — Roadmap

Git history / code archaeology TUI. Sibling of [creel](https://github.com/rsiota/creel):
one Go binary, vim-native motions, results-grid thinking, relationship navigation,
low-friction answers to “how did this code get here?”

**Product stance:** archaeology-only (read-only). Staging/commit belongs to lazygit/gitui;
ore stays focused on exploration.

Priorities below are suggestions. Each wave should leave something usable.

---

## Core mental model

Treat the repository as a navigable graph of **commits ↔ files ↔ hunks/lines ↔ authors/branches**,
not just a linear log. “Follow this change through time and across renames” should feel as
fluid as following foreign keys in creel.

---

## Target layout (3–4 panes)

```
┌─ Sidebar ──────────┬─ Main Grid / View ──────────────────────────────┬─ Detail ─────┐
│  Branches / Tags   │  Commit list  or  File history  or  Blame grid   │  Diff /      │
│  or File tree      │  (tabular, sortable, filterable)                  │  Patch /     │
│  or Search results │                                                  │  Commit msg  │
│                    ├──────────────────────────────────────────────────┤              │
│                    │  Optional bottom: Timeline strip or Hunk list    │              │
└────────────────────┴──────────────────────────────────────────────────┴──────────────┘
```

- **Sidebar** (`alt+b`): branches/tags, file tree, authors, bookmarks, search results
- **Main area**: primary results grid
- **Detail pane**: live diff, message, stats, line context
- Status bar + command palette (`Ctrl+P`) + help (`?`) — creel-style

---

## Waves

### Wave 0 — Scaffold ✅

**Goal:** open a repo and browse commits with a live detail pane.

- [x] `git` CLI backend: `Open`, `CommitLog`, `Show`
- [x] Bubble Tea shell: commit grid + detail (message / stat / patch)
- [x] `j/k`, `g`/`G`, page, `Tab`, `q`, `?`
- [x] Light-terminal diff washes (Cursor/GitHub-style)
- [x] Binding registry → `?` help overlay
- [x] `/` filter on the current grid
- [x] Light chrome (GitHub-light title/focus/muted)
- [x] Columnar commit grid: `h`/`l` cells, `o` sort, column-scoped `/`
- [x] Soft graph column (like `git log --graph`, navigable)
- [x] Extend grid widget to files / history / blame
- [x] Command palette (`Ctrl+P`)
- [x] Zen / unified detail diff (`D`)
- [x] Themes picker / dark variant (`:theme light|dark`, persisted)

Files: `cmd/ore`, `internal/git`, `internal/ui`, `internal/config`, `internal/version`

### Wave 1 — File history ✅ (in progress)

**Goal:** from a commit file (or path), show path history including renames.

- [x] File list for selected commit → Enter opens history grid
- [x] `git log --follow -- path` (`FileHistory`)
- [x] Detail pane tracks the selected history row (path-scoped `ShowPath`)
- [x] `esc` / `backspace` to walk back files → commits
- [x] Ex commands: `:blame`, `:history`, `:goto`, `:help`, `:q`

### Wave 2 — Blame / archaeology grid ✅ (started)

**Goal:** lines as first-class rows — the distinctive mode.

| Line | Age / Commit | Author | Snippet | Subject |
|------|--------------|--------|---------|---------|
| 42   | 3d a1b2      | alice  | `if err := …` | Refactor auth… |

- [x] Blame grid from file (`b`) or history (`b` / Enter)
- [x] Soft age washes + relative age column
- [x] Cursor → detail shows commit that last touched the line (path-scoped)
- [x] `f` / `g f` follow line backward via porcelain `previous`
- [x] `:blame path` ex-command (and `:history`, `:goto`)
- [x] Author colouring (quiet hue on author column only; code stays age-washed)
- [ ] Collapse/expand logical blocks (later)
- [x] **Line evolution stack** — navigable provenance of a blame line across renames (`F` / `:evolve`); multi-step follow, not only one-hop `f`

### Wave 3 — Relationship explorer (`g r`) ✅ (started)

Analogous to creel’s FK explorer:

- [x] From **commit**: parents, children, files, author (`g r`)
- [x] From **blame line**: this commit, previous, file history, file
- [x] Docked right pane; `j/k`, `Enter` to jump, `esc`/`h` to close, `Tab` focus
- [x] Lazy nested expand / deeper graph walk (`l` expand, `h` collapse)
- [ ] Linked issues/PRs if parsed (thin message/body parse → jumpable edges)
- [x] Co-changed file hot spots (`Often with` in `g r`)
- [x] Co-change → intersecting commits (`Enter` on Often-with / `:couple`)
- [x] Pickaxe / content-history search (`:pickaxe` / `:S` string, `:G` regexp)

### Wave 4 — Polish & power

- [ ] Timeline / churn strip
- [x] Session restore (repo + view + file + commit)
- [x] Bookmarks / named views (commits, files, blame lines)
- [x] Readonly code yank browser (detail + blame code column)
- [ ] Charts (`M` / `:bar` churn, author frequency) — optional
- [ ] AI assist (optional, creel `Ctrl+F` shape): explain evolution, summarize range
- [ ] Packaging (GoReleaser, brew/scoop/AUR) when ready to publish

---

## Next up (sequenced)

North star: make **line → previous versions → co-changed regions → authors** feel as fluid as creel’s FK walk. Stay archaeology-only.

1. [x] **Line evolution stack** — from a blame line, walk porcelain `previous` into a scannable grid (`F`); Enter opens blame at that step; `f` stays one-hop
2. [x] **Pickaxe / content-history search** — `git log -S` / `-G` → results grid → blame / follow (`:pickaxe` / `:S` / `:G`)
3. [x] **Co-change → intersecting commits** — from Often-with (or `:couple`), list commits where paths co-occur
4. [x] **Rename / move edges everywhere** — “was X” / “moved from Y” in history, blame, and `g r` (not only `--follow`)
5. [x] **Detail hunk-walk** — `]`/`[` (and `{`/`}`) jump hunk headers in FocusDetail; zen context stays on main
5b. [x] **Bookmarks / named views** — `m` save, `g m` / `ctrl+g` / `:bookmarks` fuzzy popup; jump reuses session restore
6. [x] **Readonly code yank browser** — thin vim-like nav in panes that show code (**detail** + **blame code column**): motions (`hjkl`, `w/b/e`, `0/$`, `gg/G`, optional `f/t` + pane-local `/`), visual char/line (`v`/`V`), `y`/`Y` to clipboard; visible block cursor; clear enter/exit so it doesn’t fight grid keys (`esc` leaves). No insert, no mutating operators, no creel-scale vim engine. On blame, `Tab` enters code yank then detail.
7. [x] **Soft DAG jumps** — `p` / `c` / `u` (and `g p` / `g c` / `g u`) hop parent / first child / merge-base with tip without opening `g r`

Later (when the above feels sticky): hunk-centric mode, timeline strip, quiet ownership summary in `g r`, export provenance trail, large-repo caching. Keep AI/charts/PR links optional and thin.

---

## Interaction philosophy (creel-style)

- Vim everywhere: motions, operators where they earn their keep
- Command palette + `:` ex commands (`:blame`, `:history`, `:follow`, `:authors`, …)
- Read-only default; no write-path creep in early waves
- Dense but scannable; themes for light and dark terminals
- Charts/AI available later, never required

---

## Why not just tig / lazygit?

| Tool | Strength | Gap ore fills |
|------|----------|----------------|
| lazygit / gitui | Write path (stage, commit, branch) | Not archaeology-first |
| tig / git log | Linear browse | Weak structured “follow this code” |
| **ore** | Blame + history + rename follow + `g r` | Keyboard-fluid code archaeology |

---

## Non-goals (for now)

- Staging, committing, rebasing, conflict resolution
- Replacing `git` for scripting (maybe a thin CLI later; not the product)
- Shared library extraction with creel (copy patterns until overlap is proven)
