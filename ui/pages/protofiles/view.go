package protofiles

import (
	"image"
	"sort"
	"strings"
	"sync"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/navigator"
	"github.com/chapar-rest/chapar/ui/widgets"
)

var _ navigator.View = &View{}

type View struct {
	*ui.Base
	addButton            widget.Clickable
	addImportPath        widget.Clickable
	deleteSelectedButton widget.Clickable

	Prompt *widgets.Prompt

	searchBox *widgets.TextField
	grid      component.GridState

	checkAllBook widget.Bool

	mx            *sync.Mutex
	filterText    string
	items         []*Item
	filteredItems []*Item

	onAdd            func()
	onDelete         func(p *domain.ProtoFile)
	onDeleteSelected func(ids []string)
	onAddImportPath  func(path string)
}

func (v *View) OnEnter() {
}

func (v *View) Info() navigator.Info {
	return navigator.Info{
		ID:    "protofiles",
		Title: "Proto Files",
		Icon:  widgets.FileFolderIcon,
	}
}

type Item struct {
	Identifier string
	Path       string
	Package    string
	Services   string

	deleteButton widget.Clickable
	checkBox     widget.Bool

	p *domain.ProtoFile
}

func NewView(base *ui.Base) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)
	v := &View{
		Base:      base,
		mx:        &sync.Mutex{},
		searchBox: search,

		Prompt: widgets.NewPrompt("", "", ""),
	}

	v.searchBox.SetOnTextChange(func(text string) {
		if v.items == nil {
			return
		}

		v.Filter(text)
	})

	return v
}

func (v *View) SetItems(items []*domain.ProtoFile) {
	v.items = make([]*Item, 0, len(items))
	for _, item := range items {
		v.items = append(v.items, &Item{
			Identifier: item.MetaData.ID,
			Path:       item.Spec.Path,
			Package:    item.Spec.Package,
			Services:   strings.Join(item.Spec.Services, ","),
			p:          item,
		})
	}

	sort.Slice(v.items, func(i, j int) bool {
		return v.items[i].p.Spec.Path < v.items[j].p.Spec.Path
	})
}

func (v *View) ShowPrompt(title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...widgets.Option) {
	v.Prompt.Type = modalType
	v.Prompt.Title = title
	v.Prompt.Content = content
	v.Prompt.SetOptions(options...)
	v.Prompt.WithoutRememberBool()
	v.Prompt.SetOnSubmit(onSubmit)
	v.Prompt.Show()
}

func (v *View) HidePrompt() {
	v.Prompt.Hide()
}

func (v *View) AddItem(item *domain.ProtoFile) {
	v.items = append(v.items, &Item{
		Identifier: item.MetaData.ID,
		Path:       item.Spec.Path,
		Package:    item.Spec.Package,
		Services:   strings.Join(item.Spec.Services, ","),
		p:          item,
	})
}

func (v *View) RemoveItem(item *domain.ProtoFile) {
	for i, it := range v.items {
		if it.p.MetaData.ID == item.MetaData.ID {
			v.items = append(v.items[:i], v.items[i+1:]...)
			break
		}
	}
}

func (v *View) SetOnAdd(f func()) {
	v.onAdd = f
}

func (v *View) SetOnDelete(f func(w *domain.ProtoFile)) {
	v.onDelete = f
}

func (v *View) SetOnDeleteSelected(f func(ids []string)) {
	v.onDeleteSelected = f
}

func (v *View) SetOnAddImportPath(f func(path string)) {
	v.onAddImportPath = f
}

func (v *View) Filter(text string) {
	v.mx.Lock()
	defer v.mx.Unlock()

	v.filterText = text

	if text == "" {
		v.filteredItems = []*Item{}
		return
	}

	var items []*Item
	for _, item := range v.items {
		if strings.Contains(item.Path, text) || strings.Contains(item.Package, text) || strings.Contains(item.Services, text) {
			items = append(items, item)
		}
	}
	v.filteredItems = items
}

func (v *View) showImportPathInputModal() {
	m := modals.NewInputText("Add Import Path", "Enter absolute import path")
	v.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.AddBtn.Clicked(gtx) {
			v.onAddImportPath(m.TextField.GetText())
			v.Base.CloseModal()
		}

		if m.CloseBtn.Clicked(gtx) {
			v.Base.CloseModal()
		}

		return m.Layout(gtx, v.Theme)
	})
}

func (v *View) header(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.onAdd != nil {
		if v.addButton.Clicked(gtx) {
			v.onAdd()
		}
	}

	if v.addImportPath.Clicked(gtx) {
		v.showImportPathInputModal()
	}

	shouldShowDeleteSelected := false
	for _, item := range v.items {
		if item.checkBox.Value {
			shouldShowDeleteSelected = true
			break
		}
	}

	if shouldShowDeleteSelected {
		if v.deleteSelectedButton.Clicked(gtx) {
			var ids []string
			for _, item := range v.items {
				if item.checkBox.Value {
					ids = append(ids, item.p.MetaData.ID)
				}
			}

			if v.onDeleteSelected != nil {
				v.onDeleteSelected(ids)
			}
		}
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lb := material.Label(theme.Material(), unit.Sp(18), "Proto files and Import Paths")
					lb.Font.Weight = font.Bold
					return lb.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lb := material.Label(theme.Material(), unit.Sp(12), "Manage proto files as dependencies and import paths")
					return lb.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !shouldShowDeleteSelected {
						return layout.Dimensions{}
					}
					deleteSelectedBtn := widgets.Button(theme.Material(), &v.deleteSelectedButton, widgets.DeleteIcon, widgets.IconPositionStart, "Delete Selected")
					deleteSelectedBtn.Color = theme.ButtonTextColor
					deleteSelectedBtn.Background = theme.DeleteButtonBgColor
					return deleteSelectedBtn.Layout(gtx, theme)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					newBtn := widgets.Button(theme.Material(), &v.addImportPath, widgets.FileFolderIcon, widgets.IconPositionStart, "Add Import Path")
					newBtn.Color = theme.ButtonTextColor
					newBtn.Background = theme.ActionButtonBgColor
					return newBtn.Layout(gtx, theme)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					newBtn := widgets.Button(theme.Material(), &v.addButton, widgets.PlusIcon, widgets.IconPositionStart, "Add Proto file")
					newBtn.Color = theme.ButtonTextColor
					newBtn.Background = theme.ActionButtonBgColor
					return newBtn.Layout(gtx, theme)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(25)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(200)
					return v.searchBox.Layout(gtx, theme)
				}),
			)
		}),
	)
}

var headingText = []string{" ", "Path", "Package", "Services", " "}

func (v *View) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	items := v.items
	if v.filterText != "" {
		items = v.filteredItems
	}

	// if checkbox is clicked, check all checkboxes
	if v.checkAllBook.Update(gtx) {
		for _, item := range items {
			item.checkBox.Value = v.checkAllBook.Value
		}
	}

	// minSize := gtx.Dp(unit.Dp(50))
	inset := layout.UniformInset(unit.Dp(10))
	// Configure a label styled to be a heading.
	headingLabel := material.Body1(theme.Material(), "")
	headingLabel.Font.Weight = font.Bold
	headingLabel.Alignment = text.Start
	headingLabel.MaxLines = 1

	// Configure a label styled to be a body.
	bodyLabel := material.Body1(theme.Material(), "")
	bodyLabel.Alignment = text.Start
	bodyLabel.MaxLines = 1

	// Measure the height of a heading row.
	orig := gtx.Constraints
	gtx.Constraints.Min = image.Point{}
	macro := op.Record(gtx.Ops)
	dims := inset.Layout(gtx, headingLabel.Layout)
	_ = macro.Stop()

	chAllMacro := op.Record(gtx.Ops)
	chAll := widgets.CheckBox(theme.Material(), &v.checkAllBook, "")
	chAll.IconColor = theme.CheckBoxColor
	chAllDim := layout.UniformInset(unit.Dp(4)).Layout(gtx, chAll.Layout)
	chAllCall := chAllMacro.Stop()

	gtx.Constraints = orig

	border := widget.Border{
		Color:        theme.TableBorderColor,
		CornerRadius: unit.Dp(4),
		Width:        unit.Dp(1),
	}

	cellPadding := layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(0), Left: unit.Dp(6), Right: unit.Dp(4)}

	outerPadding := layout.Inset{Top: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(16)}

	return outerPadding.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return v.header(gtx, theme)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.Prompt.Layout(gtx, theme)
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(25)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return component.Table(theme.Material(), &v.grid).Layout(gtx, len(items), 5,
						func(axis layout.Axis, index, constraint int) int {
							colW := constraint - chAllDim.Size.Y
							infoW := colW / 6
							pathW := colW - chAllDim.Size.Y - infoW*2
							switch axis {
							case layout.Horizontal:
								switch index {
								case 0:
									return chAllDim.Size.Y
								case 1:
									return pathW
								case 2, 3:
									return infoW
								case 4:
									return chAllDim.Size.Y
								default:
									return 0
								}
							default:
								return dims.Size.Y
							}
						},
						func(gtx layout.Context, col int) layout.Dimensions {
							var dim layout.Dimensions
							if col == 0 {
								chAllCall.Add(gtx.Ops)
								dim = chAllDim
							} else {
								headingLabel.Text = headingText[col]
								dim = inset.Layout(gtx, headingLabel.Layout)
							}

							return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return dim
							})
						},
						func(gtx layout.Context, row, col int) layout.Dimensions {
							rowItem := items[row]
							background := theme.Bg
							if row%2 == 0 {
								background = widgets.Hovered(theme.Bg)
							}
							return layout.Background{}.Layout(gtx,
								func(gtx layout.Context) layout.Dimensions {
									defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
									paint.Fill(gtx.Ops, background)
									return layout.Dimensions{Size: gtx.Constraints.Min}
								},
								func(gtx layout.Context) layout.Dimensions {
									switch col {
									case 0:
										ch := material.CheckBox(theme.Material(), &rowItem.checkBox, "")
										ch.IconColor = theme.CheckBoxColor
										return layout.UniformInset(unit.Dp(4)).Layout(gtx, ch.Layout)
									case 1:
										bodyLabel.Text = rowItem.Path
									case 2:
										bodyLabel.Text = rowItem.Package
									case 3:
										bodyLabel.Text = rowItem.Services
									case 4:
										ib := widgets.IconButton{
											Icon:            widgets.DeleteIcon,
											Size:            unit.Dp(20),
											Color:           theme.TextColor,
											BackgroundColor: background,
											Clickable:       &rowItem.deleteButton,
										}

										ib.OnClick = func() {
											if v.onDelete != nil {
												v.onDelete(rowItem.p)
											}
										}

										return ib.Layout(gtx, theme)
									}

									// set background color based on row
									return cellPadding.Layout(gtx, bodyLabel.Layout)
								},
							)

						},
					)
				})
			}),
		)
	})
}
