package graphql

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
	req                   *domain.Request
	deps                  container.Deps
	dirty                 bool
	queryEd               *ui.Editor
	varsEd                *ui.Editor
	descEd                *ui.Editor
	respEd                *ui.Editor
	respHdrEd             *ui.Editor
	headers               *ui.Table
	preScript             *ui.Editor
	postScript            *ui.Editor
	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int
	pending               bool
	resultCh              chan result
	lastResp              *egress.Response
	statusText            string
	authState             container.AuthState
}

func Open(req *domain.Request, deps container.Deps) *Container {
	r := container.CopyRequest(req)
	if r.Spec.GraphQL == nil {
		r.Spec.GraphQL = &domain.GraphQLRequestSpec{URL: "https://example.com/graphql", Variables: "{}"}
	}
	g := r.Spec.GraphQL
	c := &Container{
		req:      r,
		deps:     deps,
		resultCh: make(chan result, 1),
		reqTabs: []ui.TabModel{
			{Title: "Query"}, {Title: "Variables"}, {Title: "Headers"},
			{Title: "Auth"}, {Title: "Pre"}, {Title: "Post"}, {Title: "Info"},
		},
		respTabs:   []ui.TabModel{{Title: "Response"}, {Title: "Headers"}},
		statusText: "Ready",
	}
	c.queryEd = ui.NewEditor([]byte(g.Query), highlight.Noop{})
	c.varsEd = ui.NewEditor([]byte(g.Variables), highlight.NewJSON())
	c.descEd = container.NewDescriptionEditor(r.MetaData.Description)
	c.respEd = ui.NewEditor(nil, highlight.Noop{})
	c.respHdrEd = ui.NewEditor(nil, highlight.Noop{})
	c.headers = container.NewKVTable("gql-hdr-"+r.MetaData.ID, c.markDirty)
	container.LoadKV(c.headers, g.Headers)
	c.authState = container.LoadAuthState(g.Auth)
	c.authState.AllowInherit = true
	c.authState.CollectionID = r.CollectionID
	if g.PreRequest.Type == domain.PrePostTypePython && g.PreRequest.Script != "" {
		c.preScript = ui.NewEditor([]byte(g.PreRequest.Script), highlight.Noop{})
	}
	if g.PostRequest.Type == domain.PrePostTypePython && g.PostRequest.Script != "" {
		c.postScript = ui.NewEditor([]byte(g.PostRequest.Script), highlight.Noop{})
	}
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindGraphQL }
func (c *Container) Title() string        { return domain.RequestDisplayName(c.req) }
func (c *Container) Dirty() bool {
	return c.dirty || c.queryEd.Modified() || c.varsEd.Modified() || c.descEd.Modified()
}
func (c *Container) Close() {
	c.queryEd.Close()
	c.varsEd.Close()
	c.descEd.Close()
	c.respEd.Close()
	c.respHdrEd.Close()
	if c.preScript != nil {
		c.preScript.Close()
	}
	if c.postScript != nil {
		c.postScript.Close()
	}
}
func (c *Container) markDirty() { c.dirty = true; c.deps.ReportDirty(true) }

func (c *Container) flush() {
	g := c.req.Spec.GraphQL
	g.Query = string(c.queryEd.Bytes())
	g.Variables = string(c.varsEd.Bytes())
	g.Headers = container.DumpKV(c.headers)
	c.req.MetaData.Description = string(c.descEd.Bytes())
	container.FlushAuth(&g.Auth, c.authState)
	container.FlushPreScript(&g.PreRequest, c.preScript)
	container.FlushPostScript(&g.PostRequest, c.postScript)
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
	c.queryEd.MarkSaved()
	c.varsEd.MarkSaved()
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
		c.handle(r)
	default:
	}
	if c.pending {
		ctx.Animate(30 * time.Millisecond)
	}
	th := ctx.Theme()
	id := c.req.MetaData.ID
	g := c.req.Spec.GraphQL
	splitDir := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)
	return ui.Column(
		ui.Row(
			ui.TextField("gql-url-"+id, g.URL).Placeholder("https://…/graphql").
				OnChange(func(s string) { g.URL = s; c.markDirty() }).
				OnSubmit(func(string) { c.Send() }).
				Grow(1),
			ui.Button("gql-save-"+id, ui.Text("Save")).IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			ui.Button("gql-send-"+id, ui.Text("Send")).Primary().IconStart(icons.Play).Hint("⌘↵").
				Disabled(c.pending).OnClick(c.Send),
		).Gap(th.Spacing.S).Margin(th.Spacing.XS).
			MarginTop(th.Spacing.S),
		ui.Splitter("gql-split-"+id, splitDir, c.reqPane(th), c.respPane(th, ctx)).Sizes(300, 0).Grow(1),
	).Grow(1)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	g := c.req.Spec.GraphQL
	rows := []ui.View{
		ui.Tabs("gql-req-tabs-"+id, c.reqTabs).Selected(c.reqActive).
			Closable(false).
			OnSelectItem(func(i int, _ string) { c.reqActive = i }).TabBackground(th.Background),
	}
	switch c.reqActive {
	case 1:
		rows = append(rows, ui.ViewOf(c.varsEd).Grow(1))
	case 2:
		rows = append(rows, container.HeadersPane(th, id, c.headers, c.req.CollectionID, c.deps.Catalog, c.markDirty))
	case 3:
		rows = append(rows, container.AuthForm(th, id, &g.Auth, &c.authState, c.deps.Catalog, c.markDirty))
	case 4:
		rows = append(rows, container.PreRequestPane(th, c.deps, &g.PreRequest, container.PrePostOpts{
			ID: id + "-pre", AllowPython: true,
		}, &c.preScript, c.markDirty))
	case 5:
		preview := ""
		if c.lastResp != nil {
			preview = container.PreviewPostSet(g.PostRequest.PostRequestSet, c.lastResp)
		}
		rows = append(rows, container.PostRequestPane(th, c.deps, &g.PostRequest, container.PrePostOpts{
			ID: id + "-post", AllowPython: true, AllowSetEnv: true,
			FromOptions: container.HTTPPostFromOptions(), DefaultFrom: domain.PostRequestSetFromResponseBody,
			DefaultStatus: 200,
		}, &c.postScript, preview, c.markDirty))
	case 6:
		rows = append(rows, container.InfoPane(th, id, c.req, c.descEd, func() {
			c.markDirty()
			c.deps.ReportTitle(domain.RequestDisplayName(c.req))
		}))
	default:
		rows = append(rows, ui.ViewOf(c.queryEd).Grow(1))
	}
	return ui.Column(
		ui.Column(rows...).
			Radius(th.Radius.Medium).
			Border(ui.TokenBorder, th.Stroke.Thick).Margin(th.Spacing.XS).
			BorderStyle(ui.BorderDotted).
			Padding(th.Spacing.S).
			Gap(th.Spacing.S).Grow(1),
	)
}

func (c *Container) respPane(th *theme.Theme, ctx *ui.Ctx) ui.View {
	id := c.req.MetaData.ID
	return ui.Column(
		ui.Column(
			ui.Text(c.statusText).Style(container.StatusLineStyle(c.statusText != "" && strings.HasPrefix(c.statusText, "error"), 0, c.lastResp != nil)),
			ui.Row(ui.Spacer(),
				ui.Button("gql-copy-"+id, ui.Text("Copy")).IconStart(icons.ClipboardCopy).OnClick(func() {
					ctx.Clipboard().Set(string(c.activeResp().Bytes()))
					c.deps.Toast("Copied")
				}),
			),
			ui.Tabs("gql-resp-tabs-"+id, c.respTabs).Selected(c.respActive).
				Closable(false).
				OnSelectItem(func(i int, _ string) { c.respActive = i }).TabBackground(th.Background),
			ui.ViewOf(c.activeResp()).Grow(1),
		).Radius(th.Radius.Medium).
			Border(ui.TokenBorder, th.Stroke.Thick).Margin(th.Spacing.XS).
			BorderStyle(ui.BorderDotted).
			Padding(th.Spacing.S).
			Gap(th.Spacing.S).Grow(1),
	)
}

func (c *Container) activeResp() *ui.Editor {
	if c.respActive == 1 {
		return c.respHdrEd
	}
	return c.respEd
}

func (c *Container) handle(r result) {
	c.pending = false
	if r.err != nil {
		c.statusText = r.err.Error()
		c.lastResp = nil
		c.respEd = replaceEditor(c.respEd, []byte(r.err.Error()))
		return
	}
	res := r.resp
	c.lastResp = res
	c.statusText = fmt.Sprintf("%d  %s  %d B", res.StatusCode, res.TimePassed.Round(time.Millisecond), len(res.Body))
	body := res.Body
	if len(body) == 0 {
		body = []byte(res.JSON)
	}
	c.respEd = replaceEditor(c.respEd, body)
	var hdr strings.Builder
	fmt.Fprintf(&hdr, "# --- Request Headers ---\n")
	for k, v := range res.RequestHeaders {
		fmt.Fprintf(&hdr, "%s: %s\n", k, v)
	}
	fmt.Fprintf(&hdr, "\n# --- Response Headers ---\n")
	for k, v := range res.ResponseHeaders {
		fmt.Fprintf(&hdr, "%s: %s\n", k, v)
	}
	c.respHdrEd = replaceEditor(c.respHdrEd, []byte(hdr.String()))
}

func replaceEditor(old *ui.Editor, data []byte) *ui.Editor {
	if old != nil {
		old.Close()
	}
	return ui.NewEditor(data, highlight.NewJSON())
}
