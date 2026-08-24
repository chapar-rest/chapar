package graphql

import (
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
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
	hdrEd                 *ui.Editor
	respEd                *ui.Editor
	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int
	pending               bool
	resultCh              chan result
	status                string
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
		reqTabs:  []ui.TabModel{{Title: "Query"}, {Title: "Variables"}, {Title: "Headers"}},
		respTabs: []ui.TabModel{{Title: "Response"}, {Title: "Headers"}},
		status:   "Ready",
	}
	c.queryEd = ui.NewEditor([]byte(g.Query), highlight.Noop{})
	c.varsEd = ui.NewEditor([]byte(g.Variables), highlight.NewJSON())
	c.hdrEd = ui.NewEditor([]byte(domain.KeyValuesToText(g.Headers)), highlight.Noop{})
	c.respEd = ui.NewEditor(nil, highlight.Noop{})
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindGraphQL }
func (c *Container) Title() string        { return c.req.MetaData.Name }
func (c *Container) Dirty() bool {
	return c.dirty || c.queryEd.Modified() || c.varsEd.Modified() || c.hdrEd.Modified()
}
func (c *Container) Close() {
	c.queryEd.Close()
	c.varsEd.Close()
	c.hdrEd.Close()
	c.respEd.Close()
}
func (c *Container) markDirty() { c.dirty = true; c.deps.ReportDirty(true) }

func (c *Container) flush() {
	g := c.req.Spec.GraphQL
	g.Query = string(c.queryEd.Bytes())
	g.Variables = string(c.varsEd.Bytes())
	g.Headers = domain.TextToKeyValue(string(c.hdrEd.Bytes()))
	for i := range g.Headers {
		g.Headers[i].Enable = true
	}
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
	c.hdrEd.MarkSaved()
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
	c.status = "Sending…"
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
	return ui.Column(
		ui.Row(
			ui.TextField("gql-title-"+id, c.req.MetaData.Name).OnChange(func(s string) {
				c.req.MetaData.Name = s
				c.markDirty()
				c.deps.ReportTitle(s)
			}).Width(160),
			ui.TextField("gql-url-"+id, g.URL).Placeholder("https://…/graphql").
				OnChange(func(s string) { g.URL = s; c.markDirty() }).Grow(1),
			ui.Button("gql-save-"+id, ui.Text("Save")).IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			ui.Button("gql-send-"+id, ui.Text("Send")).Primary().IconStart(icons.Play).Hint("⌘↵").
				Disabled(c.pending).OnClick(c.Send),
		).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.M),
		ui.Text(c.status).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)).PaddingXY(th.Spacing.M, 0),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("gql-split-"+id, ui.Horizontal, c.reqPane(th), c.respPane(th)).Sizes(0, 320).Grow(1),
	).Grow(1)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	ed := c.queryEd
	switch c.reqActive {
	case 1:
		ed = c.varsEd
	case 2:
		ed = c.hdrEd
	}
	return ui.Column(
		ui.Tabs("gql-req-tabs-"+id, c.reqTabs).Selected(c.reqActive).
			OnSelectItem(func(i int, _ string) { c.reqActive = i }).TabBackground(th.Background),
		ui.ViewOf(ed).Grow(1),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
}

func (c *Container) respPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	return ui.Column(
		ui.Tabs("gql-resp-tabs-"+id, c.respTabs).Selected(c.respActive).
			OnSelectItem(func(i int, _ string) { c.respActive = i }).TabBackground(th.Background),
		ui.ViewOf(c.respEd).Grow(1),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
}

func (c *Container) handle(r result) {
	c.pending = false
	if r.err != nil {
		c.status = r.err.Error()
		c.respEd.Close()
		c.respEd = ui.NewEditor([]byte(r.err.Error()), highlight.Noop{})
		return
	}
	res := r.resp
	c.status = fmt.Sprintf("%d  %s  %d B", res.StatusCode, res.TimePassed.Round(time.Millisecond), len(res.Body))
	body := res.Body
	if len(body) == 0 {
		body = []byte(res.JSON)
	}
	c.respEd.Close()
	if c.respActive == 1 {
		var b strings.Builder
		for k, v := range res.ResponseHeaders {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
		c.respEd = ui.NewEditor([]byte(b.String()), highlight.Noop{})
		return
	}
	c.respEd = ui.NewEditor(body, highlight.NewJSON())
}
