package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

type relRowKind int

const (
	relSection relRowKind = iota
	relCommit
	relFile
	relHotSpot // co-changed path; count shown as a leading column
	relOwn     // quiet ownership share row (author · count · %)
	relAuthor
	relMeta
)

// relMaxDepth caps nested expand so graph walks stay scannable.
const relMaxDepth = 6

type relNode struct {
	kind       relRowKind
	depth      int
	label      string
	hash       string
	path       string
	author     string   // ownership row author name (hue)
	count      int      // co-change hits; leading column when kind == relHotSpot
	coupleWith []string // seed paths for Often-with or Ownership drill
	selectable bool
	expandable bool
	expanded   bool
	loading    bool
	id         string
	parent     *relNode
	children   []*relNode
}

// RelExplorer is a docked relationship browser (creel-style g r) with lazy
// nested expand on commit rows.
type RelExplorer struct {
	open   bool
	title  string
	root   []*relNode // top-level visible tree roots
	cursor int        // index into visibleNodes()
	offset int
	width  int
	height int
	seq    int // expand request id
}

func (e *RelExplorer) Open()  { e.open = true }
func (e *RelExplorer) Close() { e.open = false; e.root = nil; e.cursor = 0; e.offset = 0 }
func (e RelExplorer) Opened() bool { return e.open }

func (e *RelExplorer) SetSize(w, h int) {
	e.width = w
	e.height = h
}

func (e *RelExplorer) nextID() string {
	e.seq++
	return fmt.Sprintf("n%d", e.seq)
}

func (e *RelExplorer) LoadCommit(rel git.CommitRelations) {
	e.root = buildCommitRoot(rel, e)
	e.title = " relationships"
	e.cursor = e.firstSelectableVisible()
	e.offset = 0
	e.open = true
}

func (e *RelExplorer) LoadLine(rel git.LineRelations) {
	e.root = buildLineRoot(rel, e)
	e.title = " line relationships"
	e.cursor = e.firstSelectableVisible()
	e.offset = 0
	e.open = true
}

func buildCommitRoot(rel git.CommitRelations, e *RelExplorer) []*relNode {
	short := rel.Hash
	if len(short) > 7 {
		short = short[:7]
	}
	var roots []*relNode
	roots = append(roots, &relNode{
		kind: relMeta, label: fmt.Sprintf("%s  %s", short, rel.Subject),
	})
	roots = append(roots, &relNode{
		kind: relAuthor, label: fmt.Sprintf("%s <%s>", rel.Author, rel.Email),
	})
	roots = append(roots, &relNode{kind: relMeta, label: ""})
	roots = append(roots, commitRelationBlocks(rel, 0, nil, e)...)
	return roots
}

func buildLineRoot(rel git.LineRelations, e *RelExplorer) []*relNode {
	bl := rel.Line
	var roots []*relNode
	roots = append(roots, &relNode{
		kind:  relMeta,
		label: fmt.Sprintf("line %d · %s  %s", bl.Line, bl.ShortHash, bl.Summary),
	})
	roots = append(roots, &relNode{
		kind: relMeta, depth: 1, label: truncateRunes(bl.Text, 60),
	})
	roots = append(roots, &relNode{kind: relMeta, label: ""})

	roots = append(roots, &relNode{kind: relSection, label: "This commit"})
	roots = append(roots, e.commitNode(bl.Hash, fmt.Sprintf("%s  %s", bl.ShortHash, bl.Summary), 1, nil))

	roots = append(roots, &relNode{kind: relSection, label: "Previous"})
	if rel.Previous != nil {
		label := formatRelCommit(*rel.Previous)
		prevPath := rel.PreviousPath
		if prevPath == "" {
			prevPath = bl.PreviousPath
		}
		if prevPath != "" && prevPath != rel.Path {
			label += " · moved from " + prevPath
		} else if prevPath != "" {
			label += " · was " + prevPath
		}
		roots = append(roots, e.commitNode(rel.Previous.Hash, label, 1, nil))
	} else {
		roots = append(roots, &relNode{kind: relMeta, depth: 1, label: "(none)"})
	}

	roots = append(roots, &relNode{kind: relSection, label: fmt.Sprintf("File history (%d)", len(rel.History))})
	for _, c := range rel.History {
		label := formatRelCommit(c.Commit)
		if edge := c.EdgeLabel(); edge != "" {
			label += " · " + edge
		} else if c.Path != "" && c.Path != rel.Path {
			label += " · was " + c.Path
		}
		n := e.commitNode(c.Hash, label, 1, nil)
		n.path = c.Path
		if n.path == "" {
			n.path = rel.Path
		}
		roots = append(roots, n)
	}

	roots = append(roots, &relNode{kind: relSection, label: "File"})
	fileLabel := rel.Path
	if rel.PreviousPath != "" && rel.PreviousPath != rel.Path {
		fileLabel = rel.Path + " (was " + rel.PreviousPath + ")"
	}
	roots = append(roots, &relNode{
		kind: relFile, depth: 1, selectable: true,
		path: rel.Path, hash: rel.Rev, label: fileLabel,
	})
	seeds := []string{rel.Path}
	if rel.PreviousPath != "" && rel.PreviousPath != rel.Path {
		seeds = append(seeds, rel.PreviousPath)
	}
	roots = append(roots, hotSpotNodes(rel.HotSpots, 0, nil, rel.Rev, seeds)...)
	roots = append(roots, ownershipNodes(rel.Ownership, 0, nil, rel.Rev, seeds)...)
	return roots
}

func commitRelationBlocks(rel git.CommitRelations, depth int, parent *relNode, e *RelExplorer) []*relNode {
	var out []*relNode
	out = append(out, &relNode{kind: relSection, depth: depth, parent: parent, label: fmt.Sprintf("Parents (%d)", len(rel.Parents))})
	if len(rel.Parents) == 0 {
		out = append(out, &relNode{kind: relMeta, depth: depth + 1, parent: parent, label: "(root)"})
	}
	for _, p := range rel.Parents {
		out = append(out, e.commitNode(p.Hash, formatRelCommit(p), depth+1, parent))
	}

	out = append(out, &relNode{kind: relSection, depth: depth, parent: parent, label: fmt.Sprintf("Children (%d)", len(rel.Children))})
	if len(rel.Children) == 0 {
		out = append(out, &relNode{kind: relMeta, depth: depth + 1, parent: parent, label: "(none)"})
	}
	for _, c := range rel.Children {
		out = append(out, e.commitNode(c.Hash, formatRelCommit(c), depth+1, parent))
	}

	out = append(out, &relNode{kind: relSection, depth: depth, parent: parent, label: fmt.Sprintf("Files (%d)", len(rel.Files))})
	if len(rel.Files) == 0 {
		out = append(out, &relNode{kind: relMeta, depth: depth + 1, parent: parent, label: "(none)"})
	}
	for _, f := range rel.Files {
		path := f.Path
		if f.OldPath != "" {
			path = f.Path + " (was " + f.OldPath + ")"
		}
		out = append(out, &relNode{
			kind: relFile, depth: depth + 1, parent: parent, selectable: true,
			path: f.Path, hash: rel.Hash, label: path,
		})
	}
	out = append(out, hotSpotNodes(rel.HotSpots, depth, parent, rel.Hash, seedPathsFromFiles(rel.Files))...)
	out = append(out, ownershipNodes(rel.Ownership, depth, parent, rel.Hash, seedPathsFromFiles(rel.Files))...)
	return out
}

func seedPathsFromFiles(files []git.FileChange) []string {
	var seeds []string
	for _, f := range files {
		seeds = append(seeds, f.Path)
		if f.OldPath != "" {
			seeds = append(seeds, f.OldPath)
		}
	}
	return seeds
}

// hotSpotNodes builds the "Often with" section for co-changed paths.
func hotSpotNodes(hot []git.CoChange, depth int, parent *relNode, rev string, seeds []string) []*relNode {
	if len(hot) == 0 {
		return nil
	}
	seeds = append([]string(nil), seeds...)
	var out []*relNode
	out = append(out, &relNode{
		kind: relSection, depth: depth, parent: parent,
		label: fmt.Sprintf("Often with (%d)", len(hot)),
	})
	for _, h := range hot {
		out = append(out, &relNode{
			kind: relHotSpot, depth: depth + 1, parent: parent, selectable: true,
			path: h.Path, hash: rev, count: h.Count, label: h.Path,
			coupleWith: seeds,
		})
	}
	return out
}

// ownershipNodes builds a quiet top-authors rollup; Enter drills that author.
func ownershipNodes(own []git.AuthorShare, depth int, parent *relNode, rev string, seeds []string) []*relNode {
	if len(own) == 0 {
		return nil
	}
	seeds = append([]string(nil), seeds...)
	out := []*relNode{{
		kind: relSection, depth: depth, parent: parent,
		label: "Ownership",
	}}
	for _, a := range own {
		out = append(out, &relNode{
			kind: relOwn, depth: depth + 1, parent: parent, selectable: true,
			author: a.Name, hash: rev, coupleWith: seeds,
			label: fmt.Sprintf("%s · %d · %d%%", a.Name, a.Count, a.Pct),
		})
	}
	return out
}

func (e *RelExplorer) commitNode(hash, label string, depth int, parent *relNode) *relNode {
	return &relNode{
		kind:       relCommit,
		depth:      depth,
		label:      label,
		hash:       hash,
		selectable: true,
		expandable: depth < relMaxDepth,
		id:         e.nextID(),
		parent:     parent,
	}
}

func formatRelCommit(c git.Commit) string {
	return fmt.Sprintf("%s  %s", c.ShortHash, c.Subject)
}

func (e *RelExplorer) visibleNodes() []*relNode {
	var out []*relNode
	var walk func([]*relNode)
	walk = func(nodes []*relNode) {
		for _, n := range nodes {
			if n == nil {
				continue
			}
			out = append(out, n)
			if n.expanded && len(n.children) > 0 {
				walk(n.children)
			}
		}
	}
	walk(e.root)
	return out
}

func (e *RelExplorer) firstSelectableVisible() int {
	vis := e.visibleNodes()
	for i, n := range vis {
		if n.selectable {
			return i
		}
	}
	return 0
}

func (e *RelExplorer) Selected() (relNode, bool) {
	vis := e.visibleNodes()
	if e.cursor < 0 || e.cursor >= len(vis) {
		return relNode{}, false
	}
	n := vis[e.cursor]
	if n == nil || !n.selectable {
		return relNode{}, false
	}
	return *n, true
}

func (e *RelExplorer) selectedNode() *relNode {
	vis := e.visibleNodes()
	if e.cursor < 0 || e.cursor >= len(vis) {
		return nil
	}
	return vis[e.cursor]
}

func (e *RelExplorer) move(delta int) {
	vis := e.visibleNodes()
	if len(vis) == 0 {
		return
	}
	for i := 0; i < len(vis); i++ {
		e.cursor += delta
		if e.cursor < 0 {
			e.cursor = len(vis) - 1
		}
		if e.cursor >= len(vis) {
			e.cursor = 0
		}
		if vis[e.cursor].selectable || !hasSelectableNodes(vis) {
			break
		}
		vis = e.visibleNodes()
	}
	e.ensureVisible()
}

func hasSelectableNodes(nodes []*relNode) bool {
	for _, n := range nodes {
		if n != nil && n.selectable {
			return true
		}
	}
	return false
}

func (e *RelExplorer) ensureVisible() {
	h := max(1, e.height)
	vis := e.visibleNodes()
	if e.cursor >= len(vis) {
		e.cursor = max(0, len(vis)-1)
	}
	if e.cursor < e.offset {
		e.offset = e.cursor
	}
	if e.cursor >= e.offset+h {
		e.offset = e.cursor - h + 1
	}
}

func (e *RelExplorer) cursorToNode(target *relNode) {
	vis := e.visibleNodes()
	for i, n := range vis {
		if n == target {
			e.cursor = i
			e.ensureVisible()
			return
		}
	}
}

func (e *RelExplorer) cursorToFirstChild(parent *relNode) {
	vis := e.visibleNodes()
	for i, n := range vis {
		if n.parent == parent {
			e.cursor = i
			e.ensureVisible()
			return
		}
	}
}

// ExpandOrDive expands a collapsed commit, dives into an expanded one, or
// signals activate when the row is not expandable. Returns an async load cmd
// when a nested Relations fetch is needed.
func (e *RelExplorer) ExpandOrDive() (activate bool, cmd tea.Cmd) {
	n := e.selectedNode()
	if n == nil {
		return false, nil
	}
	if n.kind == relFile || n.kind == relHotSpot || n.kind == relOwn {
		return true, nil
	}
	if n.kind != relCommit || !n.expandable {
		return n.selectable, nil
	}
	if n.expanded {
		e.cursorToFirstChild(n)
		return false, nil
	}
	if n.loading {
		return false, nil
	}
	if n.children != nil {
		n.expanded = true
		e.cursorToFirstChild(n)
		return false, nil
	}
	if n.hash == "" {
		return true, nil
	}
	if ancestorHasHash(n.parent, n.hash) {
		n.children = []*relNode{{
			kind: relMeta, depth: n.depth + 1, parent: n, label: "(already in path)",
		}}
		n.expanded = true
		e.cursorToFirstChild(n)
		return false, nil
	}
	n.loading = true
	n.children = []*relNode{{
		kind: relMeta, depth: n.depth + 1, parent: n, label: "loading…",
	}}
	n.expanded = true
	e.ensureVisible()
	id := n.id
	hash := n.hash
	return false, func() tea.Msg {
		return relExpandRequestMsg{nodeID: id, hash: hash}
	}
}

func ancestorHasHash(n *relNode, hash string) bool {
	for p := n; p != nil; p = p.parent {
		if p.hash != "" && hashMatch(p.hash, hash) {
			return true
		}
	}
	return false
}

// CollapseOrClose collapses the current subtree, or closes the explorer when
// already at a top-level collapsed row.
func (e *RelExplorer) CollapseOrClose() (closed bool) {
	n := e.selectedNode()
	if n == nil {
		e.Close()
		return true
	}
	if n.expanded {
		n.expanded = false
		e.cursorToNode(n)
		return false
	}
	if n.parent != nil {
		p := n.parent
		// Walk up to the expandable commit parent if we're on a section/meta child.
		for p != nil && !p.expandable {
			p = p.parent
		}
		if p != nil && p.expanded {
			p.expanded = false
			e.cursorToNode(p)
			return false
		}
		if p != nil {
			e.cursorToNode(p)
			return false
		}
	}
	e.Close()
	return true
}

// ApplyExpand attaches lazy-loaded relations under the matching node.
func (e *RelExplorer) ApplyExpand(nodeID string, rel git.CommitRelations, err error) {
	n := e.findNodeByID(e.root, nodeID)
	if n == nil {
		return
	}
	n.loading = false
	if err != nil {
		n.children = []*relNode{{
			kind: relMeta, depth: n.depth + 1, parent: n, label: "error: " + err.Error(),
		}}
		n.expanded = true
		e.ensureVisible()
		return
	}
	kids := commitRelationBlocks(rel, n.depth+1, n, e)
	for _, c := range kids {
		c.parent = n
	}
	n.children = kids
	n.expanded = true
	e.cursorToFirstChild(n)
}

func (e *RelExplorer) findNodeByID(nodes []*relNode, id string) *relNode {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if n.id == id {
			return n
		}
		if found := e.findNodeByID(n.children, id); found != nil {
			return found
		}
	}
	return nil
}

type relExpandRequestMsg struct {
	nodeID string
	hash   string
}

type relExpandLoadedMsg struct {
	nodeID string
	rel    git.CommitRelations
	err    error
}

// Update handles keys while the explorer is focused.
// activate=true → jump to selection; expandCmd is a lazy nested load.
func (e *RelExplorer) Update(msg tea.KeyMsg) (consumed bool, activate bool, expandCmd tea.Cmd) {
	if !e.open {
		return false, false, nil
	}
	switch msg.String() {
	case "j", "down":
		e.move(1)
		return true, false, nil
	case "k", "up":
		e.move(-1)
		return true, false, nil
	case "g", "home":
		e.cursor = e.firstSelectableVisible()
		e.ensureVisible()
		return true, false, nil
	case "G", "end":
		vis := e.visibleNodes()
		for i := len(vis) - 1; i >= 0; i-- {
			if vis[i].selectable {
				e.cursor = i
				break
			}
		}
		e.ensureVisible()
		return true, false, nil
	case "ctrl+d":
		e.move(max(1, e.height/2))
		return true, false, nil
	case "ctrl+u":
		e.move(-max(1, e.height/2))
		return true, false, nil
	case "l", "right":
		act, cmd := e.ExpandOrDive()
		return true, act, cmd
	case "enter":
		return true, true, nil
	case "h", "left":
		e.CollapseOrClose()
		return true, false, nil
	case "esc", "q":
		e.Close()
		return true, false, nil
	}
	return false, false, nil
}

func (e RelExplorer) View(focused bool) string {
	if !e.open || e.width <= 0 || e.height <= 0 {
		return ""
	}
	vis := e.visibleNodes()
	var lines []string
	h := e.height
	if h < 1 {
		h = 1
	}
	end := min(len(vis), e.offset+h)
	for i := e.offset; i < end; i++ {
		n := vis[i]
		text := renderRelNode(n)
		switch {
		case i == e.cursor && n.selectable:
			lines = append(lines, cell(styleFocus, text, e.width))
		case n.kind == relSection:
			lines = append(lines, cell(styleHeader, text, e.width))
		case n.kind == relCommit:
			// Commits are the primary spine — full cell fg.
			lines = append(lines, fitWidth(styleCell.Render(text), e.width))
		case n.kind == relOwn:
			lines = append(lines, fitWidth(renderOwnRow(n), e.width))
		case n.kind == relFile, n.kind == relHotSpot:
			// Paths are secondary; muted keeps the tree scannable.
			lines = append(lines, fitWidth(styleMuted.Render(text), e.width))
		case !n.selectable:
			lines = append(lines, fitWidth(styleMuted.Render(text), e.width))
		default:
			lines = append(lines, fitWidth(styleCell.Render(text), e.width))
		}
	}
	return padPane(lines, e.width, e.height)
}

func renderRelNode(n *relNode) string {
	indent := strings.Repeat("  ", n.depth)
	glyph := "  "
	switch {
	case n.expandable && n.expanded:
		glyph = "▾ "
	case n.expandable:
		glyph = "▸ "
	case n.selectable:
		glyph = "  "
	}
	if n.kind == relHotSpot {
		// Leading count column so frequency lines up while scanning.
		return indent + glyph + fmt.Sprintf("%3d  %s", n.count, n.label)
	}
	return indent + glyph + n.label
}

// renderOwnRow paints the author hue on the name; rest stays muted.
func renderOwnRow(n *relNode) string {
	indent := strings.Repeat("  ", n.depth)
	name := n.author
	if name == "" {
		return styleMuted.Render(indent + "  " + n.label)
	}
	rest := n.label
	if strings.HasPrefix(rest, name) {
		rest = rest[len(name):]
	}
	return indent + "  " + authorNameStyle(name).Render(name) + styleMuted.Render(rest)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
