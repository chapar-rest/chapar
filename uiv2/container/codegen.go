package container

import (
	"fmt"
	"strings"

	"github.com/chapar-rest/chapar/internal/codegen"
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/ui"
)

// ShowCodeDialog opens a modal with generated HTTP request code.
func ShowCodeDialog(c *ui.Ctx, deps Deps, req *domain.Request) {
	if req == nil || req.Spec.HTTP == nil {
		return
	}
	state := &codeDialogState{req: req, deps: deps, lang: "curl"}
	state.regenerate()
	deps.Dialogs().Show(ui.DialogOpts{
		Title:  "Generate code",
		Width:  720,
		Height: 520,
		Body: func(ctx *ui.Ctx) ui.View {
			return state.layout(ctx)
		},
		Actions: []ui.DialogAction{
			{Label: "Close"},
		},
	})
}

type codeDialogState struct {
	req    *domain.Request
	deps   Deps
	lang   string
	code   string
	editor *ui.Editor
}

func (s *codeDialogState) layout(c *ui.Ctx) ui.View {
	th := c.Theme()
	langs := []ui.SelectOption{
		{Label: "Curl", Value: "curl"},
		{Label: "Python", Value: "python"},
		{Label: "Golang", Value: "golang"},
		{Label: "Axios", Value: "axios"},
		{Label: "Node Fetch", Value: "node-fetch"},
		{Label: "Java OkHTTP", Value: "java-okhttp"},
		{Label: "Ruby Net", Value: "ruby-net"},
		{Label: ".Net", Value: "dot-net"},
	}
	if s.editor == nil {
		s.editor = ui.NewEditor([]byte(s.code), highlight.Noop{})
	}
	return ui.Column(
		ui.Row(
			ui.Select("code-lang", langs).Width(160).
				Selected(optionIndex(s.lang, langs)).
				OnChange(func(v string) { s.lang = v; s.regenerate() }),
			ui.Button("code-copy", ui.Text("Copy")).IconStart(icons.ClipboardCopy).OnClick(func() {
				c.Clipboard().Set(s.code)
				s.deps.Toast("Copied")
			}),
		).Gap(th.Spacing.S),
		ui.ViewOf(s.editor).Grow(1),
	).Gap(th.Spacing.S).Grow(1)
}

func (s *codeDialogState) regenerate() {
	var colHeaders []domain.KeyValue
	var colAuth *domain.Auth
	if s.req.CollectionID != "" && s.deps.Catalog != nil {
		if col := s.deps.Catalog.CollectionByID(s.req.CollectionID); col != nil {
			colHeaders = col.Spec.Headers
			colAuth = &col.Spec.Auth
		}
	}
	svc := codegen.DefaultService
	if env := s.deps.ActiveEnv(); env != nil {
		svc.OnActiveEnvironmentChange(env)
	}
	var (
		code string
		err  error
	)
	switch s.lang {
	case "curl":
		code, err = svc.GenerateCurlCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	case "python":
		code, err = svc.GeneratePythonRequest(s.req.Spec.HTTP, colHeaders, colAuth)
	case "golang":
		code, err = svc.GenerateGoRequest(s.req.Spec.HTTP, colHeaders, colAuth)
	case "axios":
		code, err = svc.GenerateAxiosCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	case "node-fetch":
		code, err = svc.GenerateFetchCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	case "java-okhttp":
		code, err = svc.GenerateJavaOkHttpCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	case "ruby-net":
		code, err = svc.GenerateRubyNetHttpCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	case "dot-net":
		code, err = svc.GenerateDotNetHttpClientCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	default:
		code, err = svc.GenerateCurlCommand(s.req.Spec.HTTP, colHeaders, colAuth)
	}
	if err != nil {
		code = err.Error()
	}
	s.code = code
	if s.editor != nil {
		s.editor.Close()
	}
	s.editor = ui.NewEditor([]byte(s.code), highlight.Noop{})
}

// FormatKeyValues renders key-values as properties text.
func FormatKeyValues(items []domain.KeyValue, title string) string {
	var b strings.Builder
	if title != "" {
		fmt.Fprintf(&b, "# --- %s ---\n", title)
	}
	for _, kv := range items {
		if kv.Key == "" {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", kv.Key, kv.Value)
	}
	return b.String()
}

// SplitPaneAxis returns splitter axis from horizontal-split preference.
func SplitPaneAxis(useHorizontalSplit bool) ui.Axis {
	if useHorizontalSplit {
		return ui.Vertical
	}
	return ui.Horizontal
}

// RequestBreadcrumb returns collection prefix text for a request.
func RequestBreadcrumb(req *domain.Request) string {
	if req == nil || req.CollectionName == "" {
		return ""
	}
	return req.CollectionName + " / "
}

// StatusLineStyle returns text style for a response status line.
func StatusLineStyle(err bool, code int, haveResult bool) ui.Spec {
	style := ui.Spec{}.TextColor(ui.TokenForegroundMuted)
	if err {
		style = ui.Spec{}.TextColor(ui.TokenError)
	} else if haveResult && code >= 200 && code < 300 {
		style = ui.Spec{}.TextColor(ui.TokenSuccess)
	}
	return style
}
