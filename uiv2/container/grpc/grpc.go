package grpc

import (
	"fmt"
	"strconv"
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
	// exampleBody carries a generated request body back to the UI thread.
	// Editors must not be built off it: ui.NewEditor measures text through the
	// window's shared text engine.
	exampleBody *string
}

type Container struct {
	req                   *domain.Request
	deps                  container.Deps
	dirty                 bool
	bodyEd                *ui.Editor
	descEd                *ui.Editor
	respEd                *ui.Editor
	respMetaEd            *ui.Editor
	respTrailEd           *ui.Editor
	meta                  *ui.Table
	vars                  *ui.Table
	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int
	pending               bool
	resultCh              chan result
	lastResp              *egress.Response
	statusText            string
	respRaw               bool
	timeline              container.TimelineState
	methods               []ui.SelectOption
	loading               bool
	authState             container.AuthState
	preScript             *ui.Editor
	postScript            *ui.Editor
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
		reqTabs: []ui.TabModel{
			{Title: "Body"}, {Title: "Metadata"}, {Title: "Auth"}, {Title: "Variables"},
			{Title: "Server"}, {Title: "Settings"}, {Title: "Pre"}, {Title: "Post"}, {Title: "Info"},
		},
		respTabs:   []ui.TabModel{{Title: "Response"}, {Title: "Metadata"}, {Title: "Trailers"}, {Title: "Timeline"}},
		statusText: "Ready",
	}
	c.bodyEd = ui.NewEditor([]byte(r.Spec.GRPC.Body), highlight.NewJSON())
	c.descEd = container.NewDescriptionEditor(r.MetaData.Description)
	c.respEd = ui.NewEditor(nil, highlight.Noop{})
	c.respMetaEd = ui.NewEditor(nil, highlight.Noop{})
	c.respTrailEd = ui.NewEditor(nil, highlight.Noop{})
	c.meta = container.NewKVTable("grpc-md-"+r.MetaData.ID, c.markDirty)
	c.vars = container.NewVariablesTable("grpc-vars-"+r.MetaData.ID, nil, c.markDirty)
	container.LoadKV(c.meta, r.Spec.GRPC.Metadata)
	container.LoadVariables(c.vars, r.Spec.GRPC.Variables)
	c.authState = container.LoadAuthState(r.Spec.GRPC.Auth)
	c.refreshMethods()
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindGRPC }
func (c *Container) Title() string        { return domain.RequestDisplayName(c.req) }
func (c *Container) Dirty() bool          { return c.dirty || c.bodyEd.Modified() || c.descEd.Modified() }
func (c *Container) Close() {
	c.bodyEd.Close()
	c.descEd.Close()
	c.respEd.Close()
	c.respMetaEd.Close()
	c.respTrailEd.Close()
	c.timeline.Close()
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

func (c *Container) flush() {
	c.req.Spec.GRPC.Body = string(c.bodyEd.Bytes())
	c.req.Spec.GRPC.Metadata = container.DumpKV(c.meta)
	c.req.Spec.GRPC.Variables = container.DumpVariables(c.vars)
	c.req.MetaData.Description = string(c.descEd.Bytes())
	container.FlushAuth(&c.req.Spec.GRPC.Auth, c.authState)
	container.FlushPreScript(&c.req.Spec.GRPC.PreRequest, c.preScript)
	container.FlushPostScript(&c.req.Spec.GRPC.PostRequest, c.postScript)
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
	c.statusText = "Invoking…"
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
	c.statusText = "Loading methods…"
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
			c.resultCh <- result{exampleBody: &body}
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
	splitDir := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)
	return ui.Column(
		ui.Row(
			ui.TextField("grpc-addr-"+id, spec.ServerInfo.Address).Placeholder("host:port").
				IconStart(icons.Server).
				Width(300).
				OnChange(func(s string) { spec.ServerInfo.Address = s; c.markDirty() }),
			ui.Select("grpc-method-"+id, c.methods).
				Selected(optionIndex(spec.LasSelectedMethod, c.methods)).
				OnChange(func(v string) { spec.LasSelectedMethod = v; c.markDirty() }).
				Grow(1),
			ui.Button("grpc-save-"+id, ui.Text("Save")).IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			ui.Button("grpc-send-"+id, ui.Text("Invoke")).Primary().IconStart(icons.Play).Hint("⌘↵").
				Disabled(c.pending).OnClick(c.Send),
		).Gap(th.Spacing.S).Margin(th.Spacing.XS).
			MarginTop(th.Spacing.S),
		ui.Splitter("grpc-split-"+id, splitDir, c.reqPane(th), c.respPane(th, ctx)).
			Percents(50, 50).
			HandleOnHover().
			Grow(1),
	).Grow(1)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	spec := c.req.Spec.GRPC
	rows := []ui.View{
		ui.Tabs("grpc-req-tabs-"+id, c.reqTabs).Selected(c.reqActive).
			Closable(false).
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
		rows = append(rows, container.AuthForm(th, id, &spec.Auth, &c.authState, c.deps.Catalog, c.markDirty))
	case 3:
		rows = append(rows,
			ui.Button("grpc-var-add-"+id, ui.Text("Add")).OnClick(func() { container.AddVariableRow(c.vars, 0, c.markDirty) }),
			ui.ViewOf(c.vars).Grow(1),
		)
	case 4:
		rows = append(rows, c.serverTab(th, id, spec)...)
	case 5:
		rows = append(rows, c.settingsTab(th, id, spec)...)
	case 6:
		rows = append(rows, container.PreRequestPane(th, c.deps, &spec.PreRequest, container.PrePostOpts{
			ID: id + "-pre",
		}, &c.preScript, c.markDirty))
	case 7:
		preview := ""
		if c.lastResp != nil {
			preview = container.PreviewPostSet(spec.PostRequest.PostRequestSet, c.lastResp)
		}
		rows = append(rows, container.PostRequestPane(th, c.deps, &spec.PostRequest, container.PrePostOpts{
			ID: id + "-post", AllowSetEnv: true,
			FromOptions: container.GRPCPostFromOptions(), DefaultFrom: domain.PostRequestSetFromResponseBody,
		}, &c.postScript, preview, c.markDirty))
	case 8:
		rows = append(rows, container.InfoPane(th, id, c.req, c.descEd, func() {
			c.markDirty()
			c.deps.ReportTitle(domain.RequestDisplayName(c.req))
		}))
	}
	return ui.Column(
		ui.Column(rows...).
			Padding(th.Spacing.S).
			Gap(th.Spacing.S).Grow(1),
	)
}

func (c *Container) serverTab(th *theme.Theme, id string, spec *domain.GRPCRequestSpec) []ui.View {
	protoLabel := "Choose proto file…"
	if len(spec.ServerInfo.ProtoFiles) > 0 {
		protoLabel = spec.ServerInfo.ProtoFiles[len(spec.ServerInfo.ProtoFiles)-1]
	}
	return []ui.View{
		ui.Checkbox("grpc-reflect-"+id, "Server reflection").
			Check(spec.ServerInfo.ServerReflection).
			OnToggle(func(v bool) { spec.ServerInfo.ServerReflection = v; c.markDirty() }),
		ui.Row(
			ui.Button("grpc-proto-"+id, ui.Text(protoLabel)).OnClick(func() {
				container.PickProtoFile(c.deps, func(p string) {
					spec.ServerInfo.ProtoFiles = append(spec.ServerInfo.ProtoFiles, p)
					spec.ServerInfo.ServerReflection = false
					c.markDirty()
				})
			}),
			ui.Button("grpc-load-"+id, ui.Text("Reload methods")).Disabled(c.loading).OnClick(c.loadMethods),
		).Gap(th.Spacing.S),
	}
}

func (c *Container) settingsTab(th *theme.Theme, id string, spec *domain.GRPCRequestSpec) []ui.View {
	rows := []ui.View{
		ui.Checkbox("grpc-insecure-"+id, "Insecure").
			Check(spec.Settings.Insecure).
			OnToggle(func(v bool) { spec.Settings.Insecure = v; c.markDirty() }),
		ui.TextField("grpc-timeout-"+id, strconv.Itoa(spec.Settings.TimeoutMilliseconds)).
			Placeholder("Timeout (ms)").
			OnChange(func(s string) {
				n, _ := strconv.Atoi(s)
				if n == 0 {
					n = 1000
				}
				spec.Settings.TimeoutMilliseconds = n
				c.markDirty()
			}),
	}
	if !spec.Settings.Insecure {
		rows = append(rows,
			ui.TextField("grpc-override-"+id, spec.Settings.NameOverride).Placeholder("Server name override").
				OnChange(func(s string) { spec.Settings.NameOverride = s; c.markDirty() }),
			certPicker(id, "Root cert", spec.Settings.RootCertFile, c.deps, func(p string) {
				spec.Settings.RootCertFile = p
				c.markDirty()
			}),
			certPicker(id, "Client cert", spec.Settings.ClientCertFile, c.deps, func(p string) {
				spec.Settings.ClientCertFile = p
				c.markDirty()
			}),
			certPicker(id, "Client key", spec.Settings.ClientKeyFile, c.deps, func(p string) {
				spec.Settings.ClientKeyFile = p
				c.markDirty()
			}),
		)
	}
	return rows
}

func certPicker(id, label, path string, deps container.Deps, onPick func(string)) ui.View {
	name := path
	if name == "" {
		name = "Choose " + label + "…"
	}
	return ui.Button("grpc-cert-"+label+"-"+id, ui.Text(name)).OnClick(func() {
		container.PickCertFile(deps, onPick)
	})
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
		content = container.TimelineView("grpc-tl-"+id, th, ctx, c.deps, &c.timeline, steps)
	default:
		fname := "response.txt"
		if c.lastResp != nil {
			fname = container.DefaultResponseFilename(c.lastResp.BodyKind)
		}
		content = container.ResponseEditorMenu("grpc-resp-"+id, c.activeResp(), ctx, c.deps, fname)
	}

	return ui.Column(
		ui.Column(
			ui.Text(c.statusText).Style(container.StatusLineStyle(false, 0, c.statusText != "" && c.statusText != "Ready")),
			container.ResponseTabsRow("grpc-resp-"+id, th, c.respTabs, c.respActive,
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
			Gap(th.Spacing.S).Grow(1),
	)
}

func (c *Container) activeResp() *ui.Editor {
	switch c.respActive {
	case 1:
		return c.respMetaEd
	case 2:
		return c.respTrailEd
	default:
		return c.respEd
	}
}

func (c *Container) applyBodyEditor() {
	if c.lastResp == nil {
		return
	}
	c.respEd = container.ReplaceEditor(c.respEd, container.DisplayBody(c.lastResp, c.respRaw), container.BodyHighlighter(c.lastResp.BodyKind))
}

func (c *Container) handle(r result) {
	c.pending = false
	if r.exampleBody != nil {
		c.bodyEd = container.ReplaceEditor(c.bodyEd, []byte(*r.exampleBody), highlight.NewJSON())
		c.markDirty()
		return
	}
	if r.err != nil {
		c.statusText = r.err.Error()
		c.lastResp = r.resp
		c.respEd = container.ReplaceEditor(c.respEd, []byte(r.err.Error()), highlight.Noop{})
		if r.resp != nil {
			c.timeline.SetSteps(r.resp.Timeline)
		}
		return
	}
	if r.resp == nil {
		c.statusText = "Ready"
		return
	}
	c.lastResp = r.resp
	c.statusText = fmt.Sprintf("%s  %s  %s", r.resp.Status, r.resp.TimePassed.Round(time.Millisecond), container.FormatBytes(r.resp.Size))
	c.applyBodyEditor()
	c.respMetaEd = container.ReplaceEditor(c.respMetaEd, []byte(formatMeta(r.resp)), highlight.Noop{})
	c.respTrailEd = container.ReplaceEditor(c.respTrailEd, []byte(container.FormatKeyValues(r.resp.Trailers, "Trailers")), highlight.Noop{})
	c.timeline.SetSteps(r.resp.Timeline)
	vars := container.DumpVariables(c.vars)
	container.UpdateVariablePreviews(c.vars, vars, r.resp)
}

func formatMeta(res *egress.Response) string {
	var b strings.Builder
	b.WriteString(container.FormatKeyValues(res.RequestMetadata, "Request Metadata"))
	b.WriteString("\n")
	b.WriteString(container.FormatKeyValues(res.ResponseMetadata, "Response Metadata"))
	return b.String()
}

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}
