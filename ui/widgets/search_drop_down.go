package widgets

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type SearchDropDown struct {
	menuContextArea component.ContextArea
	menu            component.MenuState
	list            *widget.List
	theme           *chapartheme.Theme

	input *TextField

	MinWidth unit.Dp
	MaxWidth unit.Dp
	menuInit bool

	isOpen              bool
	selectedOptionIndex int
	lastSelectedIndex   int
	options             []*SearchDropDownOption

	size image.Point

	borderWidth  unit.Dp
	cornerRadius unit.Dp

	onValueChange func(value string)
}

type SearchDropDownOption struct {
	Text       string
	Value      string
	Identifier string
	clickable  widget.Clickable

	Icon      *widget.Icon
	IconColor color.NRGBA
	IconSize  unit.Dp

	isDivider bool
	isDefault bool
}

func NewSearchDropDownOption(text string) *SearchDropDownOption {
	return &SearchDropDownOption{
		Text:      text,
		isDivider: false,
	}
}

func NewSearchDropDownOptionDivider() *SearchDropDownOption {
	return &SearchDropDownOption{
		isDivider: true,
	}
}

func (o *SearchDropDownOption) WithIdentifier(identifier string) *SearchDropDownOption {
	o.Identifier = identifier
	return o
}

func (o *SearchDropDownOption) WithValue(value string) *SearchDropDownOption {
	o.Value = value
	return o
}

func (o *SearchDropDownOption) WithIcon(icon *widget.Icon, color color.NRGBA, size unit.Dp) *SearchDropDownOption {
	o.Icon = icon
	o.IconColor = color
	o.IconSize = size
	return o
}

func (o *SearchDropDownOption) DefaultSelected() *SearchDropDownOption {
	o.isDefault = true
	return o
}

func (o *SearchDropDownOption) GetText() string {
	if o == nil {
		return ""
	}

	return o.Text
}

func (o *SearchDropDownOption) GetValue() string {
	if o == nil {
		return ""
	}

	return o.Value
}

func (c *SearchDropDown) SetSelected(index int) {
	c.selectedOptionIndex = index
	c.lastSelectedIndex = index
}

func (c *SearchDropDown) SetOnChanged(f func(value string)) {
	c.onValueChange = f
}

func (c *SearchDropDown) SetSelectedByTitle(title string) {
	if len(c.options) == 0 {
		return
	}

	for i, opt := range c.options {
		if opt.Text == title {
			c.selectedOptionIndex = i
			c.lastSelectedIndex = i
			break
		}
	}
}

func (c *SearchDropDown) SetSelectedByIdentifier(identifier string) {
	for i, opt := range c.options {
		if opt.Identifier == identifier {
			c.selectedOptionIndex = i
			c.lastSelectedIndex = i
			break
		}
	}
}

func (c *SearchDropDown) SetSelectedByValue(value string) {
	for i, opt := range c.options {
		if opt.Value == value {
			c.selectedOptionIndex = i
			c.lastSelectedIndex = i
			break
		}
	}
}

func NewSearchDropDown(theme *chapartheme.Theme, options ...*SearchDropDownOption) *SearchDropDown {
	search := NewTextField("", "Search...")
	search.SetIcon(SearchIcon, IconPositionEnd)
	c := &SearchDropDown{
		input: search,
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		options:      options,
		borderWidth:  unit.Dp(1),
		cornerRadius: unit.Dp(4),
		theme:        theme,
		menuInit:     true,
	}

	return c
}

func NewSearchDropDownWithoutBorder(theme *chapartheme.Theme, options ...*SearchDropDownOption) *SearchDropDown {
	c := &SearchDropDown{
		menuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		options:  options,
		theme:    theme,
		menuInit: true,
	}

	return c
}

func (c *SearchDropDown) SelectedIndex() int {
	return c.selectedOptionIndex
}

func (c *SearchDropDown) SetOptions(options ...*SearchDropDownOption) {
	c.selectedOptionIndex = 0
	c.options = options
	if len(c.options) > 0 {
		c.menuInit = true
	}
}

func (c *SearchDropDown) GetSelected() *SearchDropDownOption {
	if len(c.options) == 0 {
		return nil
	}

	return c.options[c.selectedOptionIndex]
}

func (c *SearchDropDown) SetSize(size image.Point) {
	c.size = size
}

// Layout the SearchDropDown.
func (c *SearchDropDown) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	c.isOpen = c.menuContextArea.Active()

	for i, opt := range c.options {
		if opt.isDefault {
			c.selectedOptionIndex = i
		}

		for opt.clickable.Clicked(gtx) {
			c.isOpen = false
			c.selectedOptionIndex = i
		}
	}

	if c.selectedOptionIndex != c.lastSelectedIndex {
		if c.onValueChange != nil {
			go c.onValueChange(c.options[c.selectedOptionIndex].Value)
		}
		c.lastSelectedIndex = c.selectedOptionIndex
	}

	// Update menu items only if options change
	if c.menuInit {
		c.menuInit = false
		c.updateMenuItems(theme)
	}

	if c.MinWidth == 0 {
		c.MinWidth = unit.Dp(150)
	}

	inputDims := c.input.Layout(gtx, theme)
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return inputDims
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return c.menuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				offset := layout.Inset{
					Top:  unit.Dp(float32(inputDims.Size.Y)/gtx.Metric.PxPerDp + 1),
					Left: unit.Dp(0),
				}
				return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(c.MinWidth)
					if c.MaxWidth != 0 {
						gtx.Constraints.Max.X = gtx.Dp(c.MaxWidth)
					}
					m := component.Menu(theme.Material(), &c.menu)
					m.SurfaceStyle.Fill = theme.DropDownMenuBgColor
					return m.Layout(gtx)
				})
			})
		}),
	)
}

// updateMenuItems creates or updates menu items based on options and calculates minWidth.
func (c *SearchDropDown) updateMenuItems(theme *chapartheme.Theme) {
	c.menu.Options = c.menu.Options[:0]
	for _, opt := range c.options {
		opt := opt
		c.menu.Options = append(c.menu.Options, func(gtx layout.Context) layout.Dimensions {
			if opt.isDivider {
				dv := component.Divider(theme.Material())
				dv.Fill = theme.BorderColor
				return dv.Layout(gtx)
			}

			itm := component.MenuItem(theme.Material(), &opt.clickable, opt.Text)
			if opt.Icon != nil {
				itm.Icon = opt.Icon
				itm.IconColor = opt.IconColor
				itm.IconSize = opt.IconSize
			}

			itm.Label.Color = chapartheme.White
			return itm.Layout(gtx)
		})
	}
}
