package container

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/uiv2/vars"
	"github.com/mirzakhany/yoga/ui"
)

// VarSource is the set of variables a container's fields complete and paint:
// the active environment, the built-in dynamic values, and whatever extra
// names the container knows about — the ones it writes back from a response,
// say. extra may be nil.
func VarSource(deps Deps, extra func() []vars.Entry) vars.Source {
	return vars.Source{Env: deps.ActiveEnv, Extra: extra}
}

// VarsFromTable reads the response-extraction names out of a variables table,
// so a request can complete the names it is about to define itself.
func VarsFromTable(t *ui.Table) func() []vars.Entry {
	return func() []vars.Entry {
		if t == nil {
			return nil
		}
		return vars.FromVariables(DumpVariables(t))
	}
}

// AssistKV gives a key/value table's value cells variable highlighting and
// completion. Keys are left alone: chapar substitutes values, not names.
func AssistKV(t *ui.Table, src vars.Source) {
	if t == nil {
		return
	}
	t.CellHighlight = func(_, colID, value string) []ui.TextSpan {
		if colID != kvColValue {
			return nil
		}
		return src.Highlight(value)
	}
	t.CellSuggest = func(_, colID, value string, caret int) ([]ui.Suggestion, int, int) {
		if colID != kvColValue {
			return nil, 0, 0
		}
		return src.Suggest(value, caret)
	}
	t.CellHoverInfo = func(_, colID, value string, off int) (ui.HoverCard, bool) {
		if colID != kvColValue {
			return ui.HoverCard{}, false
		}
		return src.HoverAt(value, off)
	}
}

// AssistField gives a text field variable highlighting, completion, and a
// hover card explaining the placeholder under the pointer.
func AssistField(n *ui.Node, src vars.Source) *ui.Node {
	return n.Highlight(src.Highlight).Suggest(src.Suggest).HoverInfo(src.HoverAt)
}

// AssistURLField is AssistField for an address bar, where {name} path
// parameters are marked up as well as {{name}} variables.
func AssistURLField(n *ui.Node, src vars.Source) *ui.Node {
	return n.Highlight(src.HighlightURL).Suggest(src.Suggest).HoverInfo(src.HoverAt)
}

// AssistEditor gives a code editor the same variable completion and hover as
// the single-line fields. A body or a script may hold {{name}} placeholders
// wherever text is allowed, and a language server, when one is attached, keeps
// answering for everything else.
func AssistEditor(ed *ui.Editor, src vars.Source) *ui.Editor {
	if ed == nil {
		return nil
	}
	ed.Suggest = src.Suggest
	ed.HoverInfo = src.HoverAt
	return ed
}

// ParamEntries lists a table's keys as path-parameter entries.
func ParamEntries(t *ui.Table) []vars.Entry {
	if t == nil {
		return nil
	}
	return vars.FromKeys(vars.KindParam, DumpKV(t))
}

// EnvEntries lists an environment's own enabled values, for the environment
// editor, where there is no active environment to read from.
func EnvEntries(values []domain.KeyValue) []vars.Entry {
	return vars.FromKeys(vars.KindEnv, values)
}
