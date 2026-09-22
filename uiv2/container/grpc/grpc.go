package grpc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	grpcsvc "github.com/chapar-rest/chapar/internal/egress/grpc"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/uiv2/container"
	"github.com/chapar-rest/chapar/uiv2/vars"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

type result struct {
	resp *egress.Response
	err  error
	// missing carries unresolved proto imports back to the UI thread, where
	// the dialog that asks the user to locate them can be opened.
	missing []grpcsvc.MissingImport
	// exampleBody carries a generated request body back to the UI thread.
	// Editors must not be built off it: ui.NewEditor measures text through the
	// window's shared text engine.
	exampleBody *string
}

type Container struct {
	req         *domain.Request
	deps        container.Deps
	dirty       bool
	bodyEd      *ui.Editor
	descEd      *ui.Editor
	respEd      *ui.Editor
	respMetaEd  *ui.Editor
	respTrailEd *ui.Editor
	meta        *ui.Table
	vars        *ui.Table

	// varSrc is what the address, metadata and auth fields complete and paint
	// their {{variable}} placeholders from.
	varSrc                vars.Source
	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int
	actionsNav            int
	pending               bool
	resultCh              chan result
	lastResp              *egress.Response
	statusText            string
	errText               string // error of the last failed send; shown by ErrorView
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
			{Title: "Body"}, {Title: "Metadata"}, {Title: "Auth"},
			{Title: "Server"}, {Title: "Settings"}, {Title: "Actions"}, {Title: "Info"},
		},
		respTabs:   []ui.TabModel{{Title: "Response"}, {Title: "Metadata"}, {Title: "Trailers"}, {Title: "Timeline"}},
		statusText: "Ready",
	}
	c.bodyEd = container.NewBodyEditor(deps, "grpc-body-"+r.MetaData.ID, domain.RequestBodyTypeJSON, []byte(r.Spec.GRPC.Body))
	c.descEd = container.NewDescriptionEditor(r.MetaData.Description)
	c.respEd = ui.NewEditor(nil, highlight.Noop{})
	c.respMetaEd = ui.NewEditor(nil, highlight.Noop{})
	c.respTrailEd = ui.NewEditor(nil, highlight.Noop{})
	c.meta = container.NewKVTable("grpc-md-"+r.MetaData.ID, c.markDirty)
	c.vars = container.NewVariablesTable("grpc-vars-"+r.MetaData.ID, nil, c.markDirty)
	container.LoadKV(c.meta, r.Spec.GRPC.Metadata)
	container.LoadVariables(c.vars, r.Spec.GRPC.Variables)
	c.varSrc = container.VarSource(deps, container.VarsFromTable(c.vars))
	container.AssistEditor(c.bodyEd, c.varSrc)
	container.AssistKV(c.meta, c.varSrc)
	c.authState = container.LoadAuthState(r.Spec.GRPC.Auth)
	c.authState.Vars = c.varSrc
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
	if c.preScript != nil {
		c.preScript.Close()
	}
	if c.postScript != nil {
		c.postScript.Close()
	}
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
		var missingErr *grpcsvc.MissingImportsError
		if errors.As(err, &missingErr) {
			c.resultCh <- result{missing: missingErr.Missing}
		} else if err != nil {
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
			container.AssistURLField(ui.TextField("grpc-addr-"+id, spec.ServerInfo.Address), c.varSrc).
				Placeholder("host:port").
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
		rows = append(rows, container.MetadataPane(th, id, c.meta, c.req.CollectionID, c.deps.Catalog, c.markDirty))
	case 2:
		rows = append(rows, container.AuthForm(th, id, &spec.Auth, &c.authState, c.deps.Catalog, c.markDirty))
	case 3:
		rows = append(rows, c.serverTab(th, id, spec)...)
	case 4:
		rows = append(rows, c.settingsTab(th, id, spec)...)
	case 5:
		preview := ""
		if c.lastResp != nil {
			preview = container.PreviewPostSet(spec.PostRequest.PostRequestSet, c.lastResp)
		}
		rows = append(rows, container.ActionsPane(th, container.ActionsOpts{
			ID: id, Selected: &c.actionsNav, Deps: c.deps,
			Pre: &spec.PreRequest, Post: &spec.PostRequest,
			PreOpts: container.PrePostOpts{ID: id + "-pre"},
			PostOpts: container.PrePostOpts{
				ID: id + "-post", AllowSetEnv: true,
				FromOptions: container.GRPCPostFromOptions(), DefaultFrom: domain.PostRequestSetFromResponseBody,
			},
			PreScript: &c.preScript, PostScript: &c.postScript, Preview: preview,
			Vars: c.vars, DefaultVarStatus: 0, MarkDirty: c.markDirty,
		}))
	case 6:
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
	info := &spec.ServerInfo
	rows := []ui.View{
		ui.Checkbox("grpc-reflect-"+id, "Server reflection").
			Check(info.ServerReflection).
			OnToggle(func(v bool) { info.ServerReflection = v; c.markDirty() }),
	}

	if info.ServerReflection {
		rows = append(rows, ui.Muted("Methods come from the server; proto files are not needed."))
	} else {
		rows = append(rows,
			ui.Strong("Proto files"),
			ui.PathList("grpc-protos-"+id, info.ProtoFiles).
				AddLabel("Add proto files…").
				PathEmptyText("No proto files").
				OnPaths(func(paths []string) { info.ProtoFiles = paths; c.markDirty() }).
				OnAdd(func() {
					container.PickProtoFiles(c.deps, func(picked []string) {
						for _, p := range picked {
							info.ProtoFiles = appendUnique(info.ProtoFiles, p)
						}
						c.markDirty()
						c.loadMethods()
					})
				}),
			ui.Strong("Import paths"),
			ui.Muted("Roots for the dependencies your proto files import."),
			ui.PathList("grpc-imports-"+id, info.ImportPaths).
				PathIcon(icons.Folder).
				AddLabel("Add import path…").
				PathEmptyText("None; each proto file's own folder is always searched").
				OnPaths(func(paths []string) { info.ImportPaths = paths; c.markDirty() }).
				OnAdd(func() {
					container.PickFolder(c.deps, "Add import path", func(dir string) {
						info.ImportPaths = appendUnique(info.ImportPaths, dir)
						c.markDirty()
						c.loadMethods()
					})
				}),
		)
	}

	rows = append(rows, ui.Row(
		ui.Button("grpc-load-"+id, ui.Text("Reload methods")).Disabled(c.loading).OnClick(c.loadMethods),
		ui.Spacer(),
	).Gap(th.Spacing.S))
	return rows
}

// showMissingImports asks the user to locate each dependency the proto files
// import but no import path holds. Locating one adds its root and retries, so
// the dialog reappears only while something is still unresolved.
func (c *Container) showMissingImports(missing []grpcsvc.MissingImport) {
	if c.deps.Dialogs == nil {
		c.deps.ShowError(&grpcsvc.MissingImportsError{Missing: missing})
		return
	}
	dialogs := c.deps.Dialogs()
	if dialogs == nil {
		c.deps.ShowError(&grpcsvc.MissingImportsError{Missing: missing})
		return
	}

	info := &c.req.Spec.GRPC.ServerInfo
	locate := func(name string) {
		dialogs.Close()
		container.PickFolder(c.deps, "Locate "+name, func(dir string) {
			info.ImportPaths = appendUnique(info.ImportPaths, grpcsvc.ImportRootFor(dir, name))
			c.markDirty()
			c.loadMethods()
		})
	}

	dialogs.Show(ui.DialogOpts{
		Title:    "Missing proto dependencies",
		Severity: ui.DialogSeverityWarning,
		Width:    560,
		Height:   float32(200 + 44*len(missing)),
		Body: func(ctx *ui.Ctx) ui.View {
			th := ctx.Theme()
			kids := []ui.View{
				ui.Muted("These imports were not found. Point Chapar at the folder each one lives in."),
			}
			for _, m := range missing {
				name := m.Name
				kids = append(kids, ui.Row(
					ui.Icon(icons.FileQuestionMark, th.Metrics.IconSizeSM, th.ForegroundMuted),
					ui.Text(name).Ellipsis(ui.EllipsisMiddle).Grow(1).Tooltip(name),
					ui.Button("grpc-locate-"+name, ui.Text("Locate…")).OnClick(func() { locate(name) }),
				).Gap(th.Spacing.S))
			}
			return ui.Scroll("grpc-missing-scroll", ui.Column(kids...).Gap(th.Spacing.S).Padding(th.Spacing.M))
		},
		Actions: []ui.DialogAction{{Label: "Close"}},
	})
}

// appendUnique adds p to paths unless it is already there.
func appendUnique(paths []string, p string) []string {
	if p == "" {
		return paths
	}
	for _, existing := range paths {
		if existing == p {
			return paths
		}
	}
	return append(paths, p)
}

// settingsTab lists the connection settings the old UI offered, each with the
// label and explanation it had there. The TLS rows only apply to a secure
// connection, so they hide while plain text is on.
func (c *Container) settingsTab(th *theme.Theme, id string, spec *domain.GRPCRequestSpec) []ui.View {
	set := &spec.Settings
	items := []ui.FormItem{
		ui.FormSwitch("grpc-insecure-"+id, "Plain text", "Connect without TLS (insecure connection)", set.Insecure, func(v bool) {
			set.Insecure = v
			c.markDirty()
		}),
		ui.FormNumber("grpc-timeout-"+id, "Timeout (ms)", "Timeout for the request in milliseconds; zero means none", float64(set.TimeoutMilliseconds), 0, 3_600_000, 100, func(v float64) {
			set.TimeoutMilliseconds = int(v)
			c.markDirty()
		}),
	}
	if !set.Insecure {
		certFile := func(key, label, desc string, path *string) ui.FormItem {
			return ui.FormFile("grpc-cert-"+key+"-"+id, label, desc, *path, container.CertFileFilters, func(p string) {
				*path = p
				c.markDirty()
			})
		}
		items = append(items,
			ui.FormText("grpc-override-"+id, "Server name override", "The name used to verify the common name in the server certificate", set.NameOverride, func(v string) {
				set.NameOverride = v
				c.markDirty()
			}),
			certFile("root", "Trusted root certificate", "x509 PEM trusted root certificate", &set.RootCertFile),
			certFile("cert", "Client certificate", "Public key for mutual TLS", &set.ClientCertFile),
			certFile("key", "Client key", "Private key for mutual TLS", &set.ClientKeyFile),
		)
	}
	return []ui.View{
		ui.Scroll("grpc-settings-scroll-"+id, ui.Form("grpc-settings-"+id, items...)).Grow(1),
	}
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
		if c.errText != "" {
			content = container.ErrorView("grpc-err-"+id, th, ctx, c.deps, c.errText)
			break
		}
		fname := "response.txt"
		if c.lastResp != nil {
			fname = container.DefaultResponseFilename(c.lastResp.BodyKind)
		}
		content = container.ResponseEditorMenu(c.activeResp(), ctx, c.deps, fname)
	}

	return ui.Column(
		ui.Column(
			ui.Text(c.statusText).Style(container.StatusLineStyle(c.errText != "", 0, c.statusText != "" && c.statusText != "Ready")),
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
	c.respEd = container.ReplaceResponseEditor(c.respEd, c.lastResp, c.respRaw)
}

func (c *Container) handle(r result) {
	c.pending = false
	if len(r.missing) > 0 {
		c.statusText = "Missing proto dependencies"
		c.showMissingImports(r.missing)
		return
	}
	if r.exampleBody != nil {
		c.bodyEd.Close()
		c.bodyEd = container.NewBodyEditor(c.deps, "grpc-body-"+c.req.MetaData.ID, domain.RequestBodyTypeJSON, []byte(*r.exampleBody), ui.WithSoftWrap(true))
		c.markDirty()
		return
	}
	c.errText = ""
	if r.err != nil {
		c.statusText = container.FailedStatus
		c.errText = r.err.Error()
		c.lastResp = r.resp
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
