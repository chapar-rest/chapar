package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
)

// FileTree is a recursive directory explorer. It is a thin wrapper over the
// generic Tree: it supplies a lazy directory Loader, file/folder icons, and maps
// activation/right-click onto file-oriented callbacks. Directory children are
// loaded on first expand so opening the app in a large repo does not stat the
// whole tree up front.
type FileTree struct {
	tree *Tree

	OnOpenFile func(path string)
	OnChange   func()
	// OnMove fires after a successful drag-and-drop move with the old and new paths.
	OnMove func(src, dst string)
	// Hidden, when set, decides which directory entries are left out; it is
	// read on every directory load. Nil hides dotfiles.
	Hidden func(name string, dir bool) bool
	// OnKey, when set, sees each key while the tree is focused, with the
	// selected entry's path ("" for none), before the tree's own navigation.
	// It returns true for keys it handled, such as copy and paste.
	OnKey func(path string, ev input.KeyEvent) bool
}

// NewFileTree builds an explorer for rootPath, eagerly reading the top level so
// something shows immediately; deeper levels load on demand.
func NewFileTree(rootPath string) *FileTree {
	ft := &FileTree{}

	root := &TreeNode{Label: filepath.Base(rootPath), Data: rootPath}
	t := NewTree(root)

	// Lazy directory listing: folders first, then files, both alphabetical;
	// dotfiles hidden to keep the demo tidy.
	t.Loader = func(n *TreeNode) []*TreeNode {
		dir, _ := n.Data.(string)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		var kids []*TreeNode
		for _, de := range entries {
			name := de.Name()
			if ft.hidden(name, de.IsDir()) {
				continue
			}
			kids = append(kids, &TreeNode{
				Label: name,
				Data:  filepath.Join(dir, name),
				Leaf:  !de.IsDir(),
			})
		}
		sort.SliceStable(kids, func(i, j int) bool {
			a, b := kids[i], kids[j]
			if a.Leaf != b.Leaf {
				return !a.Leaf // directories (non-leaf) first
			}
			return a.Label < b.Label
		})
		return kids
	}

	t.OnActivate = func(n *TreeNode) {
		if p, ok := n.Data.(string); ok && ft.OnOpenFile != nil {
			ft.OnOpenFile(p)
		}
	}
	t.OnToggle = func(_ *TreeNode) {
		if ft.OnChange != nil {
			ft.OnChange()
		}
	}

	t.OnDrop = func(ev DropEvent) {
		srcPath, ok := ev.Source.Data.(string)
		if !ok {
			return
		}
		// A nil target is the empty space below the rows: move to the tree root.
		target := ev.Target
		if target == nil {
			target = t.Root()
		}
		tgtPath, ok := target.Data.(string)
		if !ok {
			return
		}

		// Resolve destination directory.
		var dstDir string
		if ev.Pos == DropInside || target == t.Root() {
			dstDir = tgtPath
		} else {
			dstDir = filepath.Dir(tgtPath)
		}

		dstPath := filepath.Join(dstDir, filepath.Base(srcPath))
		if dstPath == srcPath {
			return
		}

		// Prevent dropping a folder into one of its own descendants.
		if strings.HasPrefix(dstDir+string(filepath.Separator), srcPath+string(filepath.Separator)) {
			return
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return
		}

		// Reload the directories that changed on disk.
		dstParent := target.Parent()
		if ev.Pos == DropInside || target == t.Root() {
			dstParent = target
		}
		ft.reloadNode(ev.Source.Parent())
		if dstParent != ev.Source.Parent() {
			ft.reloadNode(dstParent)
		}
		t.Rebuild()

		if ft.OnMove != nil {
			ft.OnMove(srcPath, dstPath)
		}
		if ft.OnChange != nil {
			ft.OnChange()
		}
	}

	// Re-seed now that the Loader is set (NewTree loaded the empty root before
	// the Loader existed).
	t.SetRoot(root)

	ft.tree = t
	return ft
}

func (ft *FileTree) Layout(c *Ctx) *layout.Element {
	return ft.tree.Layout(c)
}

// Tree exposes the underlying generic tree for advanced customization (icons,
// context menu, etc.).
func (ft *FileTree) Tree() *Tree { return ft.tree }

// ContentHeight is the total pixel height of all visible rows.
func (ft *FileTree) ContentHeight() float32 { return ft.tree.ContentHeight() }

// Update drives the tree's scrollbars; call once per frame after layout.
func (ft *FileTree) Update(m *input.Mouse) { ft.tree.Update(m) }

// SetFilter restricts visible rows to files/folders whose names match query.
// See Tree.SetFilter for lazy-loading limitations.
func (ft *FileTree) SetFilter(query string) { ft.tree.SetFilter(query) }

// Focus grants keyboard focus to the file tree.
func (ft *FileTree) Focus() { ft.tree.Focus() }

// Blur removes keyboard focus from the file tree.
func (ft *FileTree) Blur() { ft.tree.Blur() }

// Focused reports whether the file tree has keyboard focus.
func (ft *FileTree) Focused() bool { return ft.tree.Focused() }

// HandleText is a no-op; the file tree does not accept text input.
func (ft *FileTree) HandleText(runes []rune) { ft.tree.HandleText(runes) }

// HandleKeys processes keyboard navigation for the file tree.
func (ft *FileTree) HandleKeys(keys []input.KeyEvent) {
	if ft.OnKey == nil || !ft.tree.Focused() {
		ft.tree.HandleKeys(keys)
		return
	}
	path := ""
	if n := ft.tree.Selected(); n != nil {
		path, _ = n.Data.(string)
	}
	rest := keys[:0:0]
	for _, ev := range keys {
		if !ft.OnKey(path, ev) {
			rest = append(rest, ev)
		}
	}
	ft.tree.HandleKeys(rest)
}

// CapturesTab reports that plain Tab should move focus rather than act on the tree.
func (ft *FileTree) CapturesTab() bool { return false }

// FocusOnClick reports that clicking the tree should grant focus.
func (ft *FileTree) FocusOnClick() bool { return true }

// FocusEl returns the element used for click-to-focus hit testing.
func (ft *FileTree) FocusEl() *layout.Element { return ft.tree.FocusEl() }

// SetContextMenu installs a builder for the right-click menu. path is the
// clicked file or folder, or "" for the empty space below the rows.
func (ft *FileTree) SetContextMenu(fn func(path string) []MenuItem) {
	ft.tree.ContextMenu = func(n *TreeNode) []MenuItem {
		if n == nil {
			return fn("")
		}
		p, _ := n.Data.(string)
		return fn(p)
	}
}

// SetLabelColor colors file and folder names, for example by git status.
// fn returns false to keep the default color.
func (ft *FileTree) SetLabelColor(fn func(path string, dir bool) (render.Color, bool)) {
	if fn == nil {
		ft.tree.LabelColor = nil
		return
	}
	ft.tree.LabelColor = func(n *TreeNode) (render.Color, bool) {
		p, _ := n.Data.(string)
		return fn(p, n.branch())
	}
}

// SetIconFor picks the glyph and its color for each row, for example by file
// type. open is whether a folder is expanded. fn nil restores the defaults.
func (ft *FileTree) SetIconFor(fn func(path string, dir, open bool) (icons.Icon, render.Color)) {
	if fn == nil {
		ft.tree.IconFor = nil
		return
	}
	ft.tree.IconFor = func(n *TreeNode, expanded bool) (icons.Icon, render.Color) {
		p, _ := n.Data.(string)
		return fn(p, n.branch(), expanded)
	}
}

// CollapseAll closes every open folder and then fires OnChange.
func (ft *FileTree) CollapseAll() {
	ft.tree.CollapseAll()
	if ft.OnChange != nil {
		ft.OnChange()
	}
}

func (ft *FileTree) hidden(name string, dir bool) bool {
	if ft.Hidden != nil {
		return ft.Hidden(name, dir)
	}
	return len(name) > 0 && name[0] == '.'
}

// Root returns the directory the tree shows.
func (ft *FileTree) Root() string {
	p, _ := ft.tree.root.Data.(string)
	return p
}

// Reload re-reads the directory at path from disk after something changed
// it. Children that still exist keep their nodes, so expanded folders stay
// expanded. A path not loaded in the tree (or a file's path, which reloads its
// directory) is ignored when no loaded node matches.
func (ft *FileTree) Reload(path string) {
	n := ft.find(path)
	if n != nil && n.Leaf {
		n = n.parent
	}
	if n == nil {
		return
	}
	if !n.loaded {
		return
	}
	old := map[string]*TreeNode{}
	for _, c := range n.Children {
		if p, ok := c.Data.(string); ok {
			old[p] = c
		}
	}
	kids := ft.tree.Loader(n)
	for i, k := range kids {
		if p, ok := k.Data.(string); ok {
			if o := old[p]; o != nil && o.Leaf == k.Leaf {
				kids[i] = o
			}
		}
	}
	n.Children = kids
	for _, c := range kids {
		c.parent = n
		c.depth = n.depth + 1
	}
	ft.tree.rebuild()
}

// Reveal expands the folders down to path, loading them as needed, then
// selects and scrolls to its row. It reports false when path is outside the
// tree or does not exist.
func (ft *FileTree) Reveal(path string) bool {
	root := ft.Root()
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	n := ft.tree.root
	if rel != "." {
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			ft.tree.ensureLoaded(n)
			var next *TreeNode
			for _, c := range n.Children {
				if c.Label == part {
					next = c
					break
				}
			}
			if next == nil {
				return false
			}
			n = next
		}
	}
	return ft.tree.Reveal(n)
}

// find returns the loaded node whose path is path, or nil.
func (ft *FileTree) find(path string) *TreeNode {
	path = filepath.Clean(path)
	var walk func(n *TreeNode) *TreeNode
	walk = func(n *TreeNode) *TreeNode {
		if p, ok := n.Data.(string); ok && filepath.Clean(p) == path {
			return n
		}
		for _, c := range n.Children {
			if f := walk(c); f != nil {
				return f
			}
		}
		return nil
	}
	return walk(ft.tree.root)
}

// reloadNode clears a directory node's children and re-reads them from disk if
// the node is currently expanded. A nil node is treated as the tree root.
func (ft *FileTree) reloadNode(n *TreeNode) {
	if n == nil {
		n = ft.tree.root
	}
	n.loaded = false
	n.Children = nil
	if n.expanded {
		ft.tree.ensureLoaded(n)
	}
}
