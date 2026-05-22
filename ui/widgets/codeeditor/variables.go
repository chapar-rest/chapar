package codeeditor

import (
	"image"
	"regexp"
	"sort"
	"unicode/utf8"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/variables"
	"github.com/chapar-rest/chapar/ui/chapartheme"
)

// VariableResolver looks up the resolved value for a template variable name.
type VariableResolver func(name string) (value string, ok bool)

// VariableEntry describes a completion candidate for template variables.
type VariableEntry struct {
	Name  string
	Value string
	Kind  string
}

// VariableLister returns variables available for completion.
type VariableLister func() []VariableEntry

var (
	defaultVariableResolver VariableResolver
	defaultVariableLister VariableLister
)

// SetDefaultVariableResolver sets the resolver used by new code editors.
func SetDefaultVariableResolver(resolver VariableResolver) {
	defaultVariableResolver = resolver
}

// SetDefaultVariableLister sets the lister used for variable auto-completion.
func SetDefaultVariableLister(lister VariableLister) {
	defaultVariableLister = lister
}

// EnvironmentVariableResolver resolves {{name}} placeholders from the active
// environment and built-in dynamic variables.
func EnvironmentVariableResolver(getEnv func() *domain.Environment) VariableResolver {
	return func(name string) (string, bool) {
		if v, ok := variables.GetVariables()[name]; ok {
			return v, true
		}

		env := getEnv()
		if env == nil {
			return "", false
		}

		for _, kv := range env.Spec.Values {
			if kv.Key == name && kv.Enable {
				return kv.Value, true
			}
		}

		return "", false
	}
}

// EnvironmentVariableLister lists variables from the active environment and
// built-in dynamic variables for auto-completion.
func EnvironmentVariableLister(getEnv func() *domain.Environment) VariableLister {
	return func() []VariableEntry {
		entries := make([]VariableEntry, 0)

		for name := range variables.GetVariables() {
			entries = append(entries, VariableEntry{
				Name:  name,
				Value: "(dynamic)",
				Kind:  "builtin",
			})
		}

		env := getEnv()
		if env != nil {
			for _, kv := range env.Spec.Values {
				if !kv.Enable {
					continue
				}
				entries = append(entries, VariableEntry{
					Name:  kv.Key,
					Value: kv.Value,
					Kind:  "env",
				})
			}
		}

		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Kind == entries[j].Kind {
				return entries[i].Name < entries[j].Name
			}
			return entries[i].Kind < entries[j].Kind
		})

		return entries
	}
}

var templateVariablePattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_$-]+)\}\}`)

type variableHover struct {
	name     string
	value    string
	defined  bool
	position image.Point
	active   bool
}

func findVariableAtRuneOffset(text string, runeOff int) (name string, ok bool) {
	for _, loc := range templateVariablePattern.FindAllStringSubmatchIndex(text, -1) {
		if len(loc) < 4 {
			continue
		}

		start := utf8.RuneCountInString(text[:loc[0]])
		end := utf8.RuneCountInString(text[:loc[1]])
		if runeOff < start || runeOff >= end {
			continue
		}

		return text[loc[2]:loc[3]], true
	}

	return "", false
}

func layoutVariableTooltip(gtx layout.Context, theme *chapartheme.Theme, hover variableHover) layout.Dimensions {
	mat := theme.Material()

	subtitle := hover.name
	value := hover.value
	if hover.defined {
		if value == "" {
			value = "(empty)"
		}
	} else {
		subtitle = hover.name + " · undefined"
		value = "No value in the active environment"
	}

	corner := unit.Dp(6)
	maxWidth := gtx.Dp(unit.Dp(320))
	maxHeight := gtx.Dp(unit.Dp(200))
	pad := unit.Dp(10)
	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, maxWidth+gtx.Dp(pad)*2)
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, maxHeight)
	gtx.Constraints.Min.Y = 0

	content := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(mat, unit.Sp(12), subtitle)
				lbl.Color = theme.InfoColor
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(4)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(mat, unit.Sp(13), value)
				lbl.Color = mat.Fg
				lbl.MaxLines = 10
				return lbl.Layout(gtx)
			}),
		)
	}

	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(pad).Layout(gtx, content)
	call := macro.Stop()

	rr := clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(corner))
	defer rr.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, theme.ContrastBg)
	call.Add(gtx.Ops)
	return dims
}
