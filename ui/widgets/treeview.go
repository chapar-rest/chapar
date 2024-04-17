package widgets

import (
	"fmt"
	"image"
	"sort"
	"strings"

	"gioui.org/io/pointer"

	"gioui.org/op"

	"gioui.org/font"

	"github.com/mirzakhany/chapar/internal/safemap"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/mirzakhany/chapar/ui/chapartheme"
)

type TreeView struct {
	materialTheme *material.Theme

	nodes []*TreeNode
	list  widget.List

	onMenuItemClick func(tr *TreeNode, item string)

	filterText    string
	filteredNodes []*TreeNode

	onNodeDoubleClick func(tr *TreeNode)
	onNodeClick       func(tr *TreeNode)

	itemLabelStyle material.LabelStyle
	openIcon       *widget.Icon
	closeIcon      *widget.Icon
	menuIcon       *widget.Icon

	// cache
	nodeDims     []layout.Dimensions
	nodeOpsCall  []op.CallOp
	cacheInvalid bool

	MenuOptions     []string
	menuClickables  []widget.Clickable
	menuInitialized bool
	menuArea        component.ContextArea
	menu            component.MenuState
}

type TreeNode struct {
	Text           string
	Identifier     string
	Children       []*TreeNode
	DiscloserState component.DiscloserState
	MenuOptions    []string

	clickable widget.Clickable

	expanded bool

	Meta *safemap.Map[string]
}

func NewTreeView(nodes []*TreeNode, theme *chapartheme.Theme) *TreeView {
	// sort nodes alphabetically
	sort.Slice(nodes, func(i, j int) bool {
		sort.Slice(nodes[i].Children, func(k, l int) bool {
			return nodes[i].Children[k].Text < nodes[i].Children[l].Text
		})
		return nodes[i].Text < nodes[j].Text
	})

	th := theme.Material()
	tr := &TreeView{
		materialTheme: th,
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		nodes: nodes,

		itemLabelStyle: material.Label(th, unit.Sp(13), ""),
		openIcon:       ExpandIcon,
		closeIcon:      ForwardIcon,
		menuIcon:       MoreVertIcon,

		nodeDims:     make([]layout.Dimensions, len(nodes)),
		nodeOpsCall:  make([]op.CallOp, len(nodes)),
		cacheInvalid: true,

		menuArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
	}

	tr.itemLabelStyle.Font.Weight = font.SemiBold
	tr.itemLabelStyle.MaxLines = 1

	return tr
}

func (t *TreeView) OnNodeDoubleClick(fn func(tr *TreeNode)) {
	t.onNodeDoubleClick = fn
}

func (t *TreeView) OnNodeClick(fn func(tr *TreeNode)) {
	t.onNodeClick = fn
}

func (t *TreeView) SetNodes(nodes []*TreeNode) {
	t.nodes = nodes
	t.nodeDims = make([]layout.Dimensions, len(t.nodes))
	t.nodeOpsCall = make([]op.CallOp, len(t.nodes))
}

func (tr *TreeNode) SetIdentifier(identifier string) {
	tr.Identifier = identifier
}

func (t *TreeView) SetOnMenuItemClick(fn func(tr *TreeNode, item string)) {
	t.onMenuItemClick = fn
}

func (t *TreeView) AddNode(node *TreeNode) {
	t.nodes = append(t.nodes, node)
}

func (tr *TreeNode) AddChildNode(child *TreeNode) {
	tr.Children = append(tr.Children, child)
}

func (t *TreeView) ExpandNode(identifier string) {
	for _, n := range t.nodes {
		if n.Identifier == identifier {
			n.expanded = true
			return
		}
	}
}

func (t *TreeView) AddChildNode(parentIdentifier string, child *TreeNode) {
	for _, n := range t.nodes {
		if n.Identifier == parentIdentifier {
			n.Children = append(n.Children, child)
			return
		}
	}
	t.cacheInvalid = true
}

func (t *TreeView) RemoveNode(identifier string) {
	for i, n := range t.nodes {
		if n.Identifier == identifier {
			t.nodes = append(t.nodes[:i], t.nodes[i+1:]...)
			t.cacheInvalid = true
			return
		}

		for j, c := range n.Children {
			if c.Identifier == identifier {
				n.Children = append(n.Children[:j], n.Children[j+1:]...)
				t.cacheInvalid = true
				return
			}
		}
	}
}

func (t *TreeView) Filter(text string) {
	t.filterText = text

	if text == "" {
		t.filteredNodes = make([]*TreeNode, 0)
		return
	}

	var items = make([]*TreeNode, 0)
	for _, item := range t.nodes {
		if strings.Contains(item.Text, text) {
			items = append(items, item)
		}

		for _, child := range item.Children {
			if strings.Contains(child.Text, text) {
				items = append(items, child)
			}
		}
	}

	t.cacheInvalid = true
	t.filteredNodes = items
}

func (t *TreeView) itemLayout(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode, isChildNode bool) layout.Dimensions {
	leftPadding := 4
	if len(node.Children) == 0 {
		leftPadding = 20
	}

	if isChildNode {
		leftPadding = 32
	}

	// apparently split view is covering the right side of the list, so we need to add padding to the right
	// to make sure menu tree-view menu icon is visible
	padding := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8 + leftPadding), Right: unit.Dp(20)}

	for {
		click, ok := node.clickable.Update(gtx)
		if !ok {
			break
		}
		switch click.NumClicks {
		case 1:
			node.DiscloserState.Appear(gtx.Now)
			if t.onNodeClick != nil {
				go t.onNodeClick(node)
			}
		case 2:
			if t.onNodeDoubleClick != nil {
				go t.onNodeDoubleClick(node)
			}
		default:
			if node.Children == nil {
				continue
			}
		}
	}

	if node.expanded {
		node.expanded = false
		node.DiscloserState.Appear(gtx.Now)
	}

	//macro := op.Record(gtx.Ops)
	//gtx.Constraints.Min.X = gtx.Dp(16)
	//gtx.Constraints.Max.X = gtx.Dp(16)
	//menuIconDims := t.menuIcon.Layout(gtx, theme.ContrastFg)
	//iconCall := macro.Stop()

	itemLayout := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}
	return node.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				background := theme.Palette.Bg
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
				if gtx.Source == (input.Source{}) {
					background = Disabled(theme.Palette.Bg)
				} else if node.clickable.Hovered() || gtx.Focused(node.clickable) {
					background = Hovered(theme.Palette.Bg)
				}

				paint.Fill(gtx.Ops, background)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return padding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return itemLayout.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							t.itemLabelStyle.Text = node.Text
							return t.itemLabelStyle.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !node.clickable.Hovered() && !t.menuArea.Active() {
								return layout.Dimensions{}
							}
							gtx.Constraints.Min.X = gtx.Dp(16)
							menuIconDims := t.menuIcon.Layout(gtx, theme.ContrastFg)
							return layout.Stack{}.Layout(gtx,
								layout.Stacked(func(gtx layout.Context) layout.Dimensions {
									return menuIconDims
								}),
								layout.Expanded(func(gtx layout.Context) layout.Dimensions {
									return t.menuArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										offset := layout.Inset{
											Top:  unit.Dp(float32(menuIconDims.Size.Y)/gtx.Metric.PxPerDp + 1),
											Left: unit.Dp(4),
										}
										return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min = image.Point{}
											m := component.Menu(theme.Material(), &t.menu)
											m.SurfaceStyle.Fill = theme.MenuBgColor
											return m.Layout(gtx)
										})
									})
								}),
							)
						}),
					)
				})
			},
		)
	})
}

// LayoutTreeNode recursively lays out a tree of widgets described by
// TreeNodes.
func (t *TreeView) layoutTreeNode(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode, isChildNode bool) layout.Dimensions {
	if len(node.Children) == 0 {
		return t.itemLayout(gtx, theme, node, isChildNode)
	}

	children := make([]layout.FlexChild, 0, len(node.Children))
	for i := range node.Children {
		children = append(children, layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				return t.layoutTreeNode(gtx, theme, node.Children[i], true)
			}))
	}

	return component.Discloser(t.materialTheme, &node.DiscloserState).Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(16.4)
			if node.DiscloserState.Visible() {
				return t.openIcon.Layout(gtx, theme.ContrastFg)
			} else {
				return t.closeIcon.Layout(gtx, theme.ContrastFg)
			}
		},
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return t.itemLayout(gtx, theme, node, isChildNode)
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		},
	)
}

func (t *TreeView) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	t.initMenus()

	nodes := t.nodes
	if t.filterText != "" {
		nodes = t.filteredNodes
	}

	if len(nodes) == 0 {
		return layout.Center.Layout(gtx, material.Label(theme.Material(), unit.Sp(14), "No items").Layout)
	}

	t.updateNodeCache(gtx, nodes, theme)

	return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(t.materialTheme, &t.list).Layout(gtx, len(nodes), func(gtx layout.Context, index int) layout.Dimensions {
			//return t.layoutTreeNode(gtx, theme, nodes[index], false)
			t.nodeOpsCall[index].Add(gtx.Ops)
			return t.nodeDims[index]
		})
	})
}

func (t *TreeView) updateNodeCache(gtx layout.Context, nodes []*TreeNode, theme *chapartheme.Theme) {
	for i := range nodes {
		m := op.Record(gtx.Ops)
		gtx.Constraints.Min = image.Point{X: gtx.Constraints.Max.X}
		t.nodeDims[i] = t.layoutTreeNode(gtx, theme, nodes[i], false)
		t.nodeOpsCall[i] = m.Stop()
	}
}

func (t *TreeView) initMenus() {
	if t.menuInitialized {
		return
	}

	fmt.Println("initMenus")

	menuOptions := make([]func(gtx layout.Context) layout.Dimensions, 0)
	t.menuClickables = make([]widget.Clickable, 0)

	for i, item := range t.MenuOptions {
		i := i
		t.menuClickables = append(t.menuClickables, widget.Clickable{})
		menuOptions = append(menuOptions, component.MenuItem(t.materialTheme, &t.menuClickables[i], item).Layout)
	}

	t.menu = component.MenuState{Options: menuOptions}
	t.menuInitialized = true
}
