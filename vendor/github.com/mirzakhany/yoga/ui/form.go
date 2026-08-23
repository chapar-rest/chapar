package ui

import (
	"fmt"
	"strconv"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
)

// FormKind selects the control shown on the right of a form row.
type FormKind int

const (
	FormItemSwitch FormKind = iota
	FormItemSelect
	FormItemNumber
	FormItemText
	FormItemSlider
	FormItemStepper
)

// FormItem is one labeled settings row.
type FormItem struct {
	ID, Label, Description string
	Icon                         icons.Icon
	Kind                         FormKind
	// Switch
	Checked  bool
	OnToggle func(bool)
	// Select
	Options  []SelectOption
	Selected int
	OnChange func(string)
	// Number / text
	Text     string
	Number   float64
	Min, Max float64
	Step     float64
	OnNumber func(float64)
	OnText   func(string)
}

// FormSwitch builds a switch row.
func FormSwitch(id, label, desc string, on bool, fn func(bool)) FormItem {
	return FormItem{ID: id, Label: label, Description: desc, Kind: FormItemSwitch, Checked: on, OnToggle: fn}
}

// FormSelect builds a select row.
func FormSelect(id, label, desc string, opts []SelectOption, selected int, fn func(string)) FormItem {
	return FormItem{ID: id, Label: label, Description: desc, Kind: FormItemSelect, Options: opts, Selected: selected, OnChange: fn}
}

// FormNumber builds a numeric text-field row.
func FormNumber(id, label, desc string, value, min, max, step float64, fn func(float64)) FormItem {
	return FormItem{ID: id, Label: label, Description: desc, Kind: FormItemNumber, Number: value, Min: min, Max: max, Step: step, OnNumber: fn}
}

// FormText builds a text-field row.
func FormText(id, label, desc, value string, fn func(string)) FormItem {
	return FormItem{ID: id, Label: label, Description: desc, Kind: FormItemText, Text: value, OnText: fn}
}

// FormSlider builds a slider row.
func FormSlider(id, label, desc string, value, min, max, step float64, fn func(float64)) FormItem {
	return FormItem{ID: id, Label: label, Description: desc, Kind: FormItemSlider, Number: value, Min: min, Max: max, Step: step, OnNumber: fn}
}

// FormStepper builds a number-stepper row.
func FormStepper(id, label, desc string, value, min, max, step float64, fn func(float64)) FormItem {
	return FormItem{ID: id, Label: label, Description: desc, Kind: FormItemStepper, Number: value, Min: min, Max: max, Step: step, OnNumber: fn}
}

type formData struct {
	items []FormItem
}

// Form renders a vertical list of labeled setting rows.
func Form(id string, items ...FormItem) *Node {
	return &Node{kind: kindForm, id: id, extra: &formData{items: items}}
}

func (n *Node) layoutForm(c *Ctx) *layout.Element {
	d, _ := n.extra.(*formData)
	if d == nil {
		d = &formData{}
	}
	th := c.Theme()
	rows := make([]View, 0, len(d.items))
	for _, item := range d.items {
		rows = append(rows, n.formRow(c, item))
	}
	return Column(rows...).Gap(th.Spacing.S).Style(n.spec).Layout(c)
}

func (n *Node) formRow(c *Ctx, item FormItem) View {
	th := c.Theme()
	pad := th.Spacing.M
	iconSz := th.Metrics.IconSizeMD

	var lead View
	if !item.Icon.Empty() {
		lead = Icon(item.Icon, iconSz, th.ForegroundMuted)
	}

	textCol := Column(
		Strong(item.Label),
		Caption(item.Description),
	).Gap(th.Spacing.XXS).Grow(1)

	control := n.formControl(c, item)

	kids := []View{textCol, control}
	if lead != nil {
		kids = append([]View{lead}, kids...)
	}

	return Row(kids...).Align(AlignStretch).Gap(th.Spacing.M).Padding(pad).
		Background(TokenSurface).
		Style(Spec{}.Radius(th.Radius.Medium).Border(TokenBorder, th.Stroke.Thin))
}

func (n *Node) formControl(c *Ctx, item FormItem) View {
	switch item.Kind {
	case FormItemSwitch:
		return Switch(item.ID).Check(item.Checked).OnToggle(item.OnToggle)
	case FormItemSelect:
		return Select(item.ID, item.Options).Width(180).Selected(item.Selected).OnChange(item.OnChange)
	case FormItemNumber:
		text := formatFormNumber(item.Number)
		return TextField(item.ID, text).Width(100).OnChange(func(s string) {
			if item.OnNumber == nil {
				return
			}
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return
			}
			if item.Min != item.Max || item.Min != 0 || item.Max != 0 {
				if v < item.Min {
					v = item.Min
				}
				if item.Max > item.Min && v > item.Max {
					v = item.Max
				}
			}
			item.OnNumber(v)
		})
	case FormItemText:
		return TextField(item.ID, item.Text).Width(180).OnChange(item.OnText)
	case FormItemSlider:
		return Slider(item.ID, item.Number).Min(item.Min).Max(item.Max).Step(item.Step).
			OnFloatChange(item.OnNumber).Width(160)
	case FormItemStepper:
		return NumberStepper(item.ID, item.Number).Min(item.Min).Max(item.Max).Step(item.Step).
			OnFloatChange(item.OnNumber)
	default:
		return Spacer()
	}
}

func formatFormNumber(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%g", v)
}
