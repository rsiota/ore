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
- [ ] Soft graph column (like `git log --graph`, navigable)
- [ ] Column sort + `/` fuzzy filter
- [ ] Binding registry → help + palette (creel pattern)
- [ ] Themes (reuse creel palette approach later)

Files: `cmd/ore`, `internal/git`, `internal/ui`, `internal/config`, `internal/version`

### Wave 1 — File history ✅ (in progress)

**Goal:** from a commit file (or path), show path history including renames.

- [x] File list for selected commit → Enter opens history grid
- [x] `git log --follow -- path` (`FileHistory`)
- [x] Detail pane tracks the selected history row (path-scoped `ShowPath`)
- [x] `esc` / `backspace` to walk back files → commits
- [ ] Ex commands: `:history`, `:open`

### Wave 2 — Blame / archaeology grid

**Goal:** lines as first-class rows — the distinctive mode.

| Line | Age / Commit | Author | Snippet | Subject |
|------|--------------|--------|---------|---------|
| 42   | 3d a1b2      | alice  | `if err := …` | Refactor auth… |

- [ ] `:blame path` / blame at revision
- [ ] Age / author colouring
- [ ] Cursor → detail shows commit + surrounding diff + mini line timeline
- [ ] `g f` / `g r` follow line backward (then forward)
- [ ] Collapse/expand logical blocks (later)

### Wave 3 — Relationship explorer (`g r`)

Analogous to creel’s FK explorer:

- From **commit**: parents, children, files, authors, linked issues/PRs if parsed
- From **file**: rename chain, co-changed files, hot spots
- From **line/hunk**: commits that shaped it
- `h/l` between levels, `Enter` to jump into that node’s view

### Wave 4 — Polish & power

- [ ] Timeline / churn strip
- [ ] Session restore (repo + view + file + commit)
- [ ] Bookmarks, authors view, search results sidebar
- [ ] Charts (`M` / `:bar` churn, author frequency) — optional
- [ ] AI assist (optional, creel `Ctrl+F` shape): explain evolution, summarize range
- [ ] Packaging (GoReleaser, brew/scoop/AUR) when ready to publish

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
