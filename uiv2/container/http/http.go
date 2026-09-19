package httpc

import (
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type result struct {
	resp *egress.Response
	err  error
}

type Container struct {
	req   *domain.Request
	deps  container.Deps
	dirty bool

	bodyEd     *ui.Editor
	descEd     *ui.Editor
	respEd     *ui.Editor
	respHdrEd  *ui.Editor
	respCkEd   *ui.Editor
	preScript  *ui.Editor
	postScript *ui.Editor

	queryParams *ui.Table
	pathParams  *ui.Table
	headers     *ui.Table
	urlEncoded  *ui.Table
	vars        *ui.Table

	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int
	actionsNav            int

	pending  bool
	resultCh chan result
	lastResp *egress.Response

	statusText string
	haveResult bool
	respCode   int
	respStatus string
	respDur    time.Duration
	respSize   int
	respErr    bool
	errText    string // error of the last failed send; shown by ErrorView
	respRaw    bool

	timeline container.TimelineState

	authState container.AuthState
}

func Open(req *domain.Request, deps container.Deps) *Container {
	r := container.CopyRequest(req)
	if r.Spec.HTTP == nil {
		r.Spec.HTTP = &domain.HTTPRequestSpec{Method: domain.RequestMethodGET, Request: &domain.HTTPRequest{}}
	}
	if r.Spec.HTTP.Request == nil {
		r.Spec.HTTP.Request = &domain.HTTPRequest{}
	}
	http := r.Spec.HTTP
	c := &Container{
		req:      r,
		deps:     deps,
		resultCh: make(chan result, 1),
		reqTabs: []ui.TabModel{
			{Title: "Params"}, {Title: "Body"}, {Title: "Auth"},
			{Title: "Headers"}, {Title: "Actions"}, {Title: "Info"},
		},
		respTabs:   []ui.TabModel{{Title: "Response"}, {Title: "Headers"}, {Title: "Cookies"}, {Title: "Timeline"}},
		statusText: "Ready",
	}
	body := ""
	if http.Request.Body.Data != "" {
		body = http.Request.Body.Data
	}
	c.bodyEd = container.NewBodyEditor(deps, "body-"+r.MetaData.ID, http.Request.Body.Type, []byte(body))
	c.descEd = container.NewDescriptionEditor(r.MetaData.Description)
	c.respEd = ui.NewEditor(nil, highlight.NewJSON(), ui.WithSoftWrap(true))
	c.respHdrEd = ui.NewEditor(nil, highlight.Noop{}, ui.WithSoftWrap(true))
	c.respCkEd = ui.NewEditor(nil, highlight.Noop{}, ui.WithSoftWrap(true))

	id := r.MetaData.ID
	c.queryParams = container.NewKVTable("query-"+id, c.markDirty)
	c.pathParams = container.NewKVTable("path-"+id, c.markDirty)
	c.headers = container.NewKVTable("hdr-"+id, c.markDirty)
	c.urlEncoded = container.NewKVTable("urlenc-"+id, c.markDirty)
	c.vars = container.NewVariablesTable("vars-"+id, nil, c.markDirty)

	container.LoadKV(c.queryParams, http.Request.QueryParams)
	container.LoadKV(c.pathParams, http.Request.PathParams)
	container.LoadKV(c.headers, http.Request.Headers)
	container.LoadKV(c.urlEncoded, http.Request.Body.URLEncoded)
	container.LoadVariables(c.vars, http.Request.Variables)

	c.authState = container.LoadAuthState(http.Request.Auth)
	c.authState.AllowInherit = true
	c.authState.CollectionID = r.CollectionID

	if http.Request.PreRequest.Type == domain.PrePostTypePython && http.Request.PreRequest.Script != "" {
		c.preScript = container.NewScriptEditor(deps, optsID(r.MetaData.ID, "pre"), http.Request.PreRequest.Script)
	}
	if http.Request.PostRequest.Type == domain.PrePostTypePython && http.Request.PostRequest.Script != "" {
		c.postScript = container.NewScriptEditor(deps, optsID(r.MetaData.ID, "post"), http.Request.PostRequest.Script)
	}
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindHTTP }
func (c *Container) Title() string        { return domain.RequestDisplayName(c.req) }
func (c *Container) Dirty() bool {
	return c.dirty || c.bodyEd.Modified() || c.descEd.Modified()
}

func (c *Container) Close() {
	c.bodyEd.Close()
	c.descEd.Close()
	c.respEd.Close()
	c.respHdrEd.Close()
	c.respCkEd.Close()
	c.timeline.Close()
	if c.preScript != nil {
		c.preScript.Close()
	}
	if c.postScript != nil {
		c.postScript.Close()
	}
}

func (c *Container) markDirty() {
	c.dirty = true
	c.deps.ReportDirty(true)
}

func (c *Container) flush() {
	http := c.req.Spec.HTTP
	http.Request.Body.Data = string(c.bodyEd.Bytes())
	http.Request.Headers = container.DumpKV(c.headers)
	http.Request.QueryParams = container.DumpKV(c.queryParams)
	http.Request.PathParams = container.DumpKV(c.pathParams)
	http.Request.Body.URLEncoded = container.DumpKV(c.urlEncoded)
	http.Request.Variables = container.DumpVariables(c.vars)
	c.req.MetaData.Description = string(c.descEd.Bytes())
	container.FlushAuth(&http.Request.Auth, c.authState)
	container.FlushPreScript(&http.Request.PreRequest, c.preScript)
	container.FlushPostScript(&http.Request.PostRequest, c.postScript)
}

func (c *Container) Save() error {
	c.flush()
	var col *domain.Collection
	if c.req.CollectionID != "" && c.deps.Catalog != nil {
		col = c.deps.Catalog.CollectionByID(c.req.CollectionID)
	}
	if err := c.deps.Repo.UpdateRequest(c.req, col); err != nil {
		return err
	}
	c.dirty = false
	c.bodyEd.MarkSaved()
	c.descEd.MarkSaved()
	c.deps.ReportDirty(false)
	if c.deps.Report.Saved != nil {
		c.deps.Report.Saved()
	}
	c.deps.Toast("Request saved")
	return nil
}

func (c *Container) Send() {
	if c.pending {
		return
	}
	c.flush()
	c.pending = true
	c.statusText = "Sending…"
	req := container.CopyRequest(c.req)
	env := c.deps.ActiveEnv()
	go func() {
		resp, err := c.deps.Sender.Send(req, env)
		c.resultCh <- result{resp: resp, err: err}
		c.deps.WakeNow()
	}()
}

func (c *Container) Layout(ctx *ui.Ctx) ui.View {
	select {
	case r := <-c.resultCh:
		c.handleResult(r)
	default:
	}
	if c.pending {
		ctx.Animate(30 * time.Millisecond)
	}

	th := ctx.Theme()
	http := c.req.Spec.HTTP
	id := c.req.MetaData.ID
	methods := methodOptions()
	splitDir := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)

	return ui.Column(
		ui.Row(
			ui.Select("http-method-"+id, methods).
				Width(110).
				Selected(optionIndex(http.Method, methods)).
				OnChange(func(v string) { http.Method = v; c.markDirty() }),
			ui.TextField("http-url-"+id, http.URL).
				Placeholder("https://…").
				OnChange(func(s string) {
					http.URL = s
					q, p := container.SyncParamsFromURL(s, container.DumpKV(c.pathParams))
					container.LoadKV(c.queryParams, q)
					container.LoadKV(c.pathParams, p)
					c.markDirty()
				}).
				OnSubmit(func(string) { c.Send() }).
				Grow(1),
			ui.Button("http-code-"+id, ui.Text("Code")).IconStart(icons.Code).OnClick(func() {
				container.ShowCodeDialog(ctx, c.deps, c.req)
			}),
			ui.Button("http-save-"+id, ui.Text("Save")).IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			ui.Button("http-send-"+id, ui.Text("Send")).Primary().Hint("⌘↵").IconStart(icons.Play).
				Disabled(c.pending).OnClick(c.Send),
		).Gap(th.Spacing.S).Margin(th.Spacing.XS).
			MarginTop(th.Spacing.S),
		ui.Splitter("http-split-"+id, splitDir, c.reqPane(th), c.respPane(th, ctx)).
			Percents(50, 50).
			HandleOnHover().
			Grow(1),
	).Grow(1)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	http := c.req.Spec.HTTP
	body := []ui.View{
		ui.Tabs("http-req-tabs-"+id, c.reqTabs).
			Selected(c.reqActive).
			Closable(false).
			OnSelectItem(func(i int, _ string) { c.reqActive = i }).
			TabBackground(th.Background),
	}
	switch c.reqActive {
	case 0:
		body = append(body, c.paramsTab(th, id)...)
	case 1:
		body = append(body, c.bodyTab(th, id, http)...)
	case 2:
		body = append(body, container.AuthForm(th, id, &http.Request.Auth, &c.authState, c.deps.Catalog, c.markDirty))
	case 3:
		body = append(body, container.HeadersPane(th, id, c.headers, c.req.CollectionID, c.deps.Catalog, c.markDirty))
	case 4:
		preview := ""
		if c.lastResp != nil {
			preview = container.PreviewPostSet(http.Request.PostRequest.PostRequestSet, c.lastResp)
		}
		body = append(body, container.ActionsPane(th, container.ActionsOpts{
			ID: id, Selected: &c.actionsNav, Deps: c.deps,
			Pre: &http.Request.PreRequest, Post: &http.Request.PostRequest,
			PreOpts: container.PrePostOpts{ID: optsID(id, "pre"), AllowPython: true},
			PostOpts: container.PrePostOpts{
				ID: optsID(id, "post"), AllowPython: true, AllowSetEnv: true,
				FromOptions: container.HTTPPostFromOptions(), DefaultFrom: domain.PostRequestSetFromResponseBody,
				DefaultStatus: 200,
			},
			PreScript: &c.preScript, PostScript: &c.postScript, Preview: preview,
			Vars: c.vars, DefaultVarStatus: 200, MarkDirty: c.markDirty,
		}))
	case 5:
		body = append(body, container.InfoPane(th, id, c.req, c.descEd, func() {
			c.markDirty()
			c.deps.ReportTitle(domain.RequestDisplayName(c.req))
		}))
	}
	return ui.Column(
		ui.Column(body...).
			Padding(th.Spacing.S).
			Gap(th.Spacing.S).Grow(1),
	)
}

func (c *Container) paramsTab(th *theme.Theme, id string) []ui.View {
	return []ui.View{
		ui.Row(ui.Text("Query"), ui.Spacer(),
			ui.IconButton("http-query-add-"+id, icons.Plus).OnClick(func() {
				container.AddKVRow(c.queryParams, func() {
					http := c.req.Spec.HTTP
					http.URL = container.SyncURLFromParams(http.URL, container.DumpKV(c.queryParams))
					c.markDirty()
				})
			}),
		).PaddingXY(0, th.Spacing.XS),
		ui.ViewOf(c.queryParams).Height(60),
		ui.Row(ui.Text("Path"), ui.Spacer(),
			ui.IconButton("http-path-add-"+id, icons.Plus).OnClick(func() {
				container.AddKVRow(c.pathParams, c.markDirty)
			}),
		).PaddingXY(0, th.Spacing.XS),
		ui.Caption("path params inside bracket, for example: {id}"),
		ui.ViewOf(c.pathParams).Grow(1),
	}
}

func (c *Container) bodyTab(th *theme.Theme, id string, http *domain.HTTPRequestSpec) []ui.View {
	types := []ui.SelectOption{
		{Label: "None", Value: domain.RequestBodyTypeNone},
		{Label: "JSON", Value: domain.RequestBodyTypeJSON},
		{Label: "XML", Value: domain.RequestBodyTypeXML},
		{Label: "Text", Value: domain.RequestBodyTypeText},
		{Label: "Form data", Value: domain.RequestBodyTypeFormData},
		{Label: "Binary", Value: domain.RequestBodyTypeBinary},
		{Label: "Urlencoded", Value: domain.RequestBodyTypeUrlencoded},
	}
	if http.Request.Body.Type == "" {
		http.Request.Body.Type = domain.RequestBodyTypeJSON
	}
	rows := []ui.View{
		ui.Select("http-body-type-"+id, types).Width(140).
			Selected(optionIndex(http.Request.Body.Type, types)).
			OnChange(func(v string) {
				http.Request.Body.Type = v
				c.bodyEd.Close()
				c.bodyEd = container.NewBodyEditor(c.deps, "body-"+c.req.MetaData.ID, v, c.bodyEd.Bytes())
				c.markDirty()
			}),
	}
	switch http.Request.Body.Type {
	case domain.RequestBodyTypeFormData:
		rows = append(rows, container.FormDataPane(th, id, &http.Request.Body.FormData.Fields, c.deps, c.markDirty))
	case domain.RequestBodyTypeBinary:
		rows = append(rows, container.BinaryFilePicker(th, id, http.Request.Body.BinaryFilePath, c.deps, func(p string) {
			http.Request.Body.BinaryFilePath = p
			c.markDirty()
		}))
	case domain.RequestBodyTypeUrlencoded:
		rows = append(rows,
			ui.Button("urlenc-add-"+id, ui.Text("Add")).OnClick(func() { container.AddKVRow(c.urlEncoded, c.markDirty) }),
			ui.ViewOf(c.urlEncoded).Grow(1),
		)
	case domain.RequestBodyTypeNone:
	default:
		rows = append(rows, ui.ViewOf(c.bodyEd).Grow(1))
	}
	return rows
}

func (c *Container) respPane(th *theme.Theme, ctx *ui.Ctx) ui.View {
	id := c.req.MetaData.ID

	var content ui.View
	switch c.respActive {
	case 3:
		var steps []egress.TimelineStep
		if c.lastResp != nil {
			steps = c.lastResp.Timeline
		}
		content = container.TimelineView("http-tl-"+id, th, ctx, c.deps, &c.timeline, steps)
	default:
		if c.respErr {
			content = container.ErrorView("http-err-"+id, th, ctx, c.deps, c.errText)
			break
		}
		fname := "response.txt"
		if c.lastResp != nil {
			fname = container.DefaultResponseFilename(c.lastResp.BodyKind)
		}
		content = container.ResponseEditorMenu("http-resp-"+id, c.activeResp(), ctx, c.deps, fname)
	}

	return ui.Column(
		ui.Column(
			c.statusLine(th),
			container.ResponseTabsRow("http-resp-"+id, th, c.respTabs, c.respActive,
				func(i int, _ string) { c.respActive = i },
				c.respRaw,
				func(raw bool) {
					c.respRaw = raw
					c.applyBodyEditor()
				},
			),
			content,
		).Radius(th.Radius.Medium).
			Border(ui.TokenBorder, th.Stroke.Thick).Margin(th.Spacing.XS).
			BorderStyle(ui.BorderDotted).
			Padding(th.Spacing.S).
			Gap(th.Spacing.XS).Grow(1),
	)
}

func (c *Container) activeResp() *ui.Editor {
	switch c.respActive {
	case 1:
		return c.respHdrEd
	case 2:
		return c.respCkEd
	default:
		return c.respEd
	}
}

func (c *Container) statusLine(th *theme.Theme) ui.View {
	return ui.Text(c.statusText).Style(container.StatusLineStyle(c.respErr, c.respCode, c.haveResult))
}

func (c *Container) applyBodyEditor() {
	if c.lastResp == nil {
		return
	}
	c.respEd = container.ReplaceResponseEditor(c.respEd, c.lastResp, c.respRaw)
}

func (c *Container) handleResult(r result) {
	c.pending = false
	c.haveResult = true
	err := r.err
	if err == nil && r.resp != nil {
		err = r.resp.Error
	}
	if err != nil {
		c.respErr = true
		c.lastResp = r.resp
		c.statusText = container.FailedStatus
		c.errText = err.Error()
		if r.resp != nil {
			c.timeline.SetSteps(r.resp.Timeline)
		}
		return
	}
	res := r.resp
	c.lastResp = res
	c.respErr = false
	c.respCode = res.StatusCode
	if c.respCode == 0 {
		c.respCode = res.StatueCode
	}
	c.respStatus = res.Status
	c.respDur = res.TimePassed
	c.respSize = len(res.Body)
	c.statusText = fmt.Sprintf("%d %s  %s  %s", c.respCode, strings.TrimSpace(c.respStatus), c.respDur.Round(time.Millisecond), container.FormatBytes(c.respSize))
	c.applyBodyEditor()
	var hdr strings.Builder
	fmt.Fprintf(&hdr, "# --- Request Headers ---\n")
	for k, v := range res.RequestHeaders {
		fmt.Fprintf(&hdr, "%s: %s\n", k, v)
	}
	fmt.Fprintf(&hdr, "\n# --- Response Headers ---\n")
	for k, v := range res.ResponseHeaders {
		fmt.Fprintf(&hdr, "%s: %s\n", k, v)
	}
	c.respHdrEd = container.ReplaceEditor(c.respHdrEd, []byte(hdr.String()), highlight.Noop{})
	var ck strings.Builder
	for _, cookie := range res.Cookies {
		fmt.Fprintf(&ck, "%s=%s\n", cookie.Name, cookie.Value)
	}
	c.respCkEd = container.ReplaceEditor(c.respCkEd, []byte(ck.String()), highlight.Noop{})
	c.timeline.SetSteps(res.Timeline)
	vars := container.DumpVariables(c.vars)
	container.UpdateVariablePreviews(c.vars, vars, res)
}

func methodOptions() []ui.SelectOption {
	out := make([]ui.SelectOption, 0, len(domain.RequestMethods))
	for _, m := range domain.RequestMethods {
		out = append(out, ui.SelectOption{Label: m, Value: m})
	}
	return out
}

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}

func optsID(id, suffix string) string { return id + "-" + suffix }
