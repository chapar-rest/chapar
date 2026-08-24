package grpc

import (
	"fmt"
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
	bodyEd                *ui.Editor
	respEd                *ui.Editor
	meta                  *ui.Table
	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int
	pending               bool
	resultCh              chan result
	status                string
	methods               []ui.SelectOption
	loading               bool
}

func Open(req *domain.Request, deps container.Deps) *Container {
	r := container.CopyRequest(req)
	if r.Spec.GRPC == nil {
		r.Spec.GRPC = &domain.GRPCRequestSpec{}
	}
	c := &Container{
		req:      r,
		deps:     deps,
		resultCh: make(chan result, 1),
		reqTabs:  []ui.TabModel{{Title: "Body"}, {Title: "Metadata"}, {Title: "Server"}},
		respTabs: []ui.TabModel{{Title: "Response"}, {Title: "Metadata"}},
		status:   "Ready",
	}
	c.bodyEd = ui.NewEditor([]byte(r.Spec.GRPC.Body), highlight.NewJSON())
	c.respEd = ui.NewEditor(nil, highlight.Noop{})
	c.meta = container.NewKVTable("grpc-md-"+r.MetaData.ID, c.markDirty)
	container.LoadKV(c.meta, r.Spec.GRPC.Metadata)
	c.refreshMethods()
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindGRPC }
func (c *Container) Title() string        { return c.req.MetaData.Name }
func (c *Container) Dirty() bool          { return c.dirty || c.bodyEd.Modified() }
func (c *Container) Close() {
	c.bodyEd.Close()
	c.respEd.Close()
}
func (c *Container) markDirty() { c.dirty = true; c.deps.ReportDirty(true) }

func (c *Container) refreshMethods() {
	c.methods = nil
	for _, svc := range c.req.Spec.GRPC.Services {
		for _, m := range svc.Methods {
			c.methods = append(c.methods, ui.SelectOption{Label: m.FullName, Value: m.FullName})
		}
	}
	if len(c.methods) == 0 {
		c.methods = []ui.SelectOption{{Label: "(load methods)", Value: ""}}
	}
}

func (c *Container) Save() error {
	c.req.Spec.GRPC.Body = string(c.bodyEd.Bytes())
	c.req.Spec.GRPC.Metadata = container.DumpKV(c.meta)
	var col *domain.Collection
	if c.req.CollectionID != "" && c.deps.Catalog != nil {
		col = c.deps.Catalog.CollectionByID(c.req.CollectionID)
	}
	if err := c.deps.Repo.UpdateRequest(c.req, col); err != nil {
		return err
	}
	c.dirty = false
	c.bodyEd.MarkSaved()
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
	c.req.Spec.GRPC.Body = string(c.bodyEd.Bytes())
	c.req.Spec.GRPC.Metadata = container.DumpKV(c.meta)
	c.pending = true
	c.status = "Invoking…"
	req := container.CopyRequest(c.req)
	env := c.deps.ActiveEnv()
	go func() {
		resp, err := c.deps.Sender.Send(req, env)
		c.resultCh <- result{resp: resp, err: err}
		c.deps.WakeNow()
	}()
}

func (c *Container) loadMethods() {
	if c.loading {
		return
	}
	c.loading = true
	c.status = "Loading methods…"
	req := container.CopyRequest(c.req)
	env := c.deps.ActiveEnv()
	go func() {
		svcs, err := c.deps.Sender.LoadGRPCServices(req, env)
		if err != nil {
			c.resultCh <- result{err: err}
		} else {
			c.req.Spec.GRPC.Services = svcs
			c.refreshMethods()
			c.resultCh <- result{}
			c.markDirty()
		}
		c.loading = false
		c.deps.WakeNow()
	}()
}

func (c *Container) loadExample() {
	req := container.CopyRequest(c.req)
	env := c.deps.ActiveEnv()
	go func() {
		body, err := c.deps.Sender.GRPCExampleBody(req, env)
		if err != nil {
			c.resultCh <- result{err: err}
		} else {
			c.bodyEd.Close()
			c.bodyEd = ui.NewEditor([]byte(body), highlight.NewJSON())
			c.markDirty()
			c.resultCh <- result{}
		}
		c.deps.WakeNow()
	}()
}

func (c *Container) Layout(ctx *ui.Ctx) ui.View {
	select {
	case r := <-c.resultCh:
		c.handle(r)
	default:
	}
	if c.pending || c.loading {
		ctx.Animate(30 * time.Millisecond)
	}
	th := ctx.Theme()
	id := c.req.MetaData.ID
	spec := c.req.Spec.GRPC
	return ui.Column(
		ui.Row(
			ui.TextField("grpc-title-"+id, c.req.MetaData.Name).OnChange(func(s string) {
				c.req.MetaData.Name = s
				c.markDirty()
				c.deps.ReportTitle(s)
			}).Width(160),
			ui.TextField("grpc-addr-"+id, spec.ServerInfo.Address).Placeholder("host:port").
				OnChange(func(s string) { spec.ServerInfo.Address = s; c.markDirty() }).Grow(1),
			ui.Select("grpc-method-"+id, c.methods).Width(260).
				Selected(optionIndex(spec.LasSelectedMethod, c.methods)).
				OnChange(func(v string) { spec.LasSelectedMethod = v; c.markDirty() }),
			ui.Button("grpc-load-"+id, ui.Text("Methods")).OnClick(c.loadMethods),
			ui.Button("grpc-save-"+id, ui.Text("Save")).IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			ui.Button("grpc-send-"+id, ui.Text("Invoke")).Primary().IconStart(icons.Play).Hint("⌘↵").
				Disabled(c.pending).OnClick(c.Send),
		).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.M),
		ui.Text(c.status).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)).PaddingXY(th.Spacing.M, 0),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("grpc-split-"+id, ui.Horizontal, c.reqPane(th), c.respPane(th)).Sizes(0, 320).Grow(1),
	).Grow(1)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	spec := c.req.Spec.GRPC
	rows := []ui.View{
		ui.Tabs("grpc-req-tabs-"+id, c.reqTabs).Selected(c.reqActive).
			OnSelectItem(func(i int, _ string) { c.reqActive = i }).TabBackground(th.Background),
	}
	switch c.reqActive {
	case 0:
		rows = append(rows,
			ui.Button("grpc-example-"+id, ui.Text("Load example")).OnClick(c.loadExample),
			ui.ViewOf(c.bodyEd).Grow(1),
		)
	case 1:
		rows = append(rows,
			ui.Button("grpc-md-add-"+id, ui.Text("Add")).OnClick(func() { container.AddKVRow(c.meta, c.markDirty) }),
			ui.ViewOf(c.meta).Grow(1),
		)
	case 2:
		rows = append(rows,
			ui.Checkbox("grpc-reflect-"+id, "Server reflection").
				Check(spec.ServerInfo.ServerReflection).
				OnToggle(func(v bool) { spec.ServerInfo.ServerReflection = v; c.markDirty() }),
			ui.Checkbox("grpc-insecure-"+id, "Insecure").
				Check(spec.Settings.Insecure).
				OnToggle(func(v bool) { spec.Settings.Insecure = v; c.markDirty() }),
		)
	}
	return ui.Column(rows...).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
}

func (c *Container) respPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	return ui.Column(
		ui.Tabs("grpc-resp-tabs-"+id, c.respTabs).Selected(c.respActive).
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
	if r.resp == nil {
		c.status = "Ready"
		return
	}
	c.status = fmt.Sprintf("%s  %s  %d B", r.resp.Status, r.resp.TimePassed.Round(time.Millisecond), r.resp.Size)
	body := r.resp.Body
	if len(body) == 0 {
		body = []byte(r.resp.JSON)
	}
	c.respEd.Close()
	c.respEd = ui.NewEditor(body, highlight.NewJSON())
}

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}
