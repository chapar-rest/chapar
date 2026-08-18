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

	bodyEd    *ui.Editor
	headersEd *ui.Editor
	respEd    *ui.Editor
	respHdrEd *ui.Editor

	params *ui.Table
	vars   *ui.Table

	reqTabs               []ui.TabModel
	respTabs              []ui.TabModel
	reqActive, respActive int

	pending  bool
	resultCh chan result

	statusText string
	haveResult bool
	respCode   int
	respStatus string
	respDur    time.Duration
	respSize   int
	respErr    bool

	authUser, authPass, authToken, authKey, authVal string
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
			{Title: "Headers"}, {Title: "Variables"}, {Title: "Pre"}, {Title: "Post"},
		},
		respTabs:   []ui.TabModel{{Title: "Response"}, {Title: "Headers"}},
		statusText: "Ready",
	}
	body := ""
	if http.Request.Body.Data != "" {
		body = http.Request.Body.Data
	}
	c.bodyEd = ui.NewEditor([]byte(body), highlight.NewJSON())
	c.headersEd = ui.NewEditor([]byte(domain.KeyValuesToText(http.Request.Headers)), highlight.Noop{})
	c.respEd = ui.NewEditor(nil, highlight.Noop{})
	c.respHdrEd = ui.NewEditor(nil, highlight.Noop{})

	c.params = container.NewKVTable("params-"+r.MetaData.ID, c.markDirty)
	container.LoadKV(c.params, append(append([]domain.KeyValue{}, http.Request.QueryParams...), http.Request.PathParams...))
	if len(http.Request.QueryParams) > 0 {
		container.LoadKV(c.params, http.Request.QueryParams)
	}

	c.vars = container.NewKVTable("vars-"+r.MetaData.ID, c.markDirty)
	varRows := make([]domain.KeyValue, 0, len(http.Request.Variables))
	for _, v := range http.Request.Variables {
		varRows = append(varRows, domain.KeyValue{ID: v.ID, Key: v.TargetEnvVariable, Value: v.JsonPath, Enable: v.Enable})
	}
	container.LoadKV(c.vars, varRows)

	if http.Request.Auth.BasicAuth != nil {
		c.authUser = http.Request.Auth.BasicAuth.Username
		c.authPass = http.Request.Auth.BasicAuth.Password
	}
	if http.Request.Auth.TokenAuth != nil {
		c.authToken = http.Request.Auth.TokenAuth.Token
	}
	if http.Request.Auth.APIKeyAuth != nil {
		c.authKey = http.Request.Auth.APIKeyAuth.Key
		c.authVal = http.Request.Auth.APIKeyAuth.Value
	}
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindHTTP }
func (c *Container) Title() string        { return c.req.MetaData.Name }
func (c *Container) Dirty() bool          { return c.dirty || c.bodyEd.Modified() || c.headersEd.Modified() }

func (c *Container) Close() {
	c.bodyEd.Close()
	c.headersEd.Close()
	c.respEd.Close()
	c.respHdrEd.Close()
}

func (c *Container) markDirty() {
	c.dirty = true
	c.deps.ReportDirty(true)
}

func (c *Container) flush() {
	http := c.req.Spec.HTTP
	http.Request.Body.Data = string(c.bodyEd.Bytes())
	http.Request.Headers = domain.TextToKeyValue(string(c.headersEd.Bytes()))
	for i := range http.Request.Headers {
		http.Request.Headers[i].Enable = true
		if http.Request.Headers[i].ID == "" {
			http.Request.Headers[i].ID = fmt.Sprintf("h-%d", i)
		}
	}
	http.Request.QueryParams = container.DumpKV(c.params)
	c.flushAuth()
}

func (c *Container) flushAuth() {
	auth := &c.req.Spec.HTTP.Request.Auth
	switch auth.Type {
	case domain.AuthTypeBasic:
		auth.BasicAuth = &domain.BasicAuth{Username: c.authUser, Password: c.authPass}
	case domain.AuthTypeToken:
		auth.TokenAuth = &domain.TokenAuth{Token: c.authToken}
	case domain.AuthTypeAPIKey:
		auth.APIKeyAuth = &domain.APIKeyAuth{Key: c.authKey, Value: c.authVal}
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
	c.bodyEd.MarkSaved()
	c.headersEd.MarkSaved()
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
	splitDir := ui.Horizontal
	if prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit {
		splitDir = ui.Vertical
	}

	return ui.Column(
		ui.Row(
			ui.TextField("http-title-"+id, c.req.MetaData.Name).OnChange(func(s string) {
				c.req.MetaData.Name = s
				c.markDirty()
				c.deps.ReportTitle(s)
			}).Width(180),
			ui.Select("http-method-"+id, methods).
				Width(110).
				Selected(optionIndex(http.Method, methods)).
				OptionColor("GET", th.Success).
				OptionColor("POST", th.Accent).
				OptionColor("PUT", th.Warning).
				OptionColor("DELETE", th.Error).
				OnChange(func(v string) { http.Method = v; c.markDirty() }),
			ui.TextField("http-url-"+id, http.URL).
				Placeholder("https://…").
				OnChange(func(s string) { http.URL = s; c.markDirty() }).
				Grow(1),
			ui.Button("http-save-"+id, ui.Text("Save")).IconStart("save").Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			ui.Button("http-send-"+id, ui.Text("Send")).Primary().Hint("⌘↵").IconStart("play_arrow").
				Disabled(c.pending).OnClick(c.Send),
		).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.M),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("http-split-"+id, splitDir, c.reqPane(th), c.respPane(th)).
			Sizes(0, 320).
			Grow(1),
	).Grow(1)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	http := c.req.Spec.HTTP
	body := []ui.View{
		ui.Tabs("http-req-tabs-"+id, c.reqTabs).
			Selected(c.reqActive).
			OnSelectItem(func(i int, _ string) { c.reqActive = i }).
			TabBackground(th.Background),
	}
	switch c.reqActive {
	case 0:
		body = append(body,
			ui.Row(
				ui.Button("http-param-add-"+id, ui.Text("Add param")).OnClick(func() {
					container.AddKVRow(c.params, c.markDirty)
				}),
			).PaddingXY(0, th.Spacing.S),
			ui.ViewOf(c.params).Grow(1),
		)
	case 1:
		types := []ui.SelectOption{
			{Label: "None", Value: domain.RequestBodyTypeNone},
			{Label: "JSON", Value: domain.RequestBodyTypeJSON},
			{Label: "XML", Value: domain.RequestBodyTypeXML},
			{Label: "Text", Value: domain.RequestBodyTypeText},
			{Label: "Binary", Value: domain.RequestBodyTypeBinary},
		}
		if http.Request.Body.Type == "" {
			http.Request.Body.Type = domain.RequestBodyTypeJSON
		}
		body = append(body,
			ui.Select("http-body-type-"+id, types).Width(140).
				Selected(optionIndex(http.Request.Body.Type, types)).
				OnChange(func(v string) { http.Request.Body.Type = v; c.markDirty() }),
			ui.ViewOf(c.bodyEd).Grow(1),
		)
	case 2:
		body = append(body, c.authForm(th))
	case 3:
		body = append(body, ui.ViewOf(c.headersEd).Grow(1))
	case 4:
		body = append(body,
			ui.Row(ui.Button("http-var-add-"+id, ui.Text("Add")).OnClick(func() {
				container.AddKVRow(c.vars, c.markDirty)
			})).PaddingXY(0, th.Spacing.S),
			ui.ViewOf(c.vars).Grow(1),
		)
	case 5:
		pre := http.Request.PreRequest
		opts := []ui.SelectOption{
			{Label: "None", Value: domain.PrePostTypeNone},
			{Label: "Trigger request", Value: domain.PrePostTypeTriggerRequest},
		}
		if pre.Type == "" {
			pre.Type = domain.PrePostTypeNone
		}
		body = append(body, ui.Select("http-pre-"+id, opts).Width(200).
			Selected(optionIndex(pre.Type, opts)).
			OnChange(func(v string) { http.Request.PreRequest.Type = v; c.markDirty() }))
		if pre.Type == domain.PrePostTypeTriggerRequest {
			body = append(body, c.triggerPick(th))
		}
	case 6:
		post := http.Request.PostRequest
		opts := []ui.SelectOption{
			{Label: "None", Value: domain.PrePostTypeNone},
			{Label: "Set environment", Value: domain.PrePostTypeSetEnv},
		}
		if post.Type == "" {
			post.Type = domain.PrePostTypeNone
		}
		body = append(body, ui.Select("http-post-"+id, opts).Width(200).
			Selected(optionIndex(post.Type, opts)).
			OnChange(func(v string) { http.Request.PostRequest.Type = v; c.markDirty() }))
		if post.Type == domain.PrePostTypeSetEnv {
			body = append(body, c.postSetForm(th))
		}
	}
	return ui.Column(body...).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
}

func (c *Container) authForm(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	auth := &c.req.Spec.HTTP.Request.Auth
	opts := []ui.SelectOption{
		{Label: "None", Value: domain.AuthTypeNone},
		{Label: "Inherit", Value: domain.AuthTypeInherit},
		{Label: "Bearer", Value: domain.AuthTypeToken},
		{Label: "Basic", Value: domain.AuthTypeBasic},
		{Label: "API Key", Value: domain.AuthTypeAPIKey},
	}
	if auth.Type == "" {
		auth.Type = domain.AuthTypeNone
	}
	rows := []ui.View{
		ui.Select("http-auth-"+id, opts).Width(180).
			Selected(optionIndex(auth.Type, opts)).
			OnChange(func(v string) { auth.Type = v; c.markDirty() }),
	}
	switch auth.Type {
	case domain.AuthTypeToken:
		rows = append(rows, ui.TextField("http-token-"+id, c.authToken).Placeholder("Token").
			OnChange(func(s string) { c.authToken = s; c.markDirty() }).Grow(1))
	case domain.AuthTypeBasic:
		rows = append(rows,
			ui.TextField("http-user-"+id, c.authUser).Placeholder("Username").
				OnChange(func(s string) { c.authUser = s; c.markDirty() }).Grow(1),
			ui.TextField("http-pass-"+id, c.authPass).Placeholder("Password").Password(true).
				OnChange(func(s string) { c.authPass = s; c.markDirty() }).Grow(1),
		)
	case domain.AuthTypeAPIKey:
		rows = append(rows,
			ui.TextField("http-apikey-"+id, c.authKey).Placeholder("Header").
				OnChange(func(s string) { c.authKey = s; c.markDirty() }).Grow(1),
			ui.TextField("http-apival-"+id, c.authVal).Placeholder("Value").
				OnChange(func(s string) { c.authVal = s; c.markDirty() }).Grow(1),
		)
	}
	return ui.Column(rows...).Gap(th.Spacing.S).Grow(1)
}

func (c *Container) triggerPick(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	if c.req.Spec.HTTP.Request.PreRequest.TriggerRequest == nil {
		c.req.Spec.HTTP.Request.PreRequest.TriggerRequest = &domain.TriggerRequest{}
	}
	tr := c.req.Spec.HTTP.Request.PreRequest.TriggerRequest
	opts := []ui.SelectOption{{Label: "None", Value: ""}}
	if c.deps.Catalog != nil {
		for _, col := range c.deps.Catalog.AllCollections() {
			for _, r := range col.Spec.Requests {
				opts = append(opts, ui.SelectOption{Label: col.MetaData.Name + " / " + r.MetaData.Name, Value: r.MetaData.ID})
			}
		}
		for _, r := range c.deps.Catalog.StandaloneRequests() {
			opts = append(opts, ui.SelectOption{Label: r.MetaData.Name, Value: r.MetaData.ID})
		}
	}
	return ui.Select("http-trigger-"+id, opts).Width(280).
		Selected(optionIndex(tr.RequestID, opts)).
		OnChange(func(v string) { tr.RequestID = v; c.markDirty() })
}

func (c *Container) postSetForm(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	set := &c.req.Spec.HTTP.Request.PostRequest.PostRequestSet
	froms := []ui.SelectOption{
		{Label: "Body", Value: domain.PostRequestSetFromResponseBody},
		{Label: "Header", Value: domain.PostRequestSetFromResponseHeader},
		{Label: "Cookie", Value: domain.PostRequestSetFromResponseCookie},
	}
	return ui.Column(
		ui.TextField("http-post-target-"+id, set.Target).Placeholder("Env key").
			OnChange(func(s string) { set.Target = s; c.markDirty() }),
		ui.TextField("http-post-fromkey-"+id, set.FromKey).Placeholder("JSONPath or header").
			OnChange(func(s string) { set.FromKey = s; c.markDirty() }),
		ui.Select("http-post-from-"+id, froms).Width(160).
			Selected(optionIndex(set.From, froms)).
			OnChange(func(v string) { set.From = v; c.markDirty() }),
	).Gap(th.Spacing.S)
}

func (c *Container) respPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	return ui.Column(
		c.statusLine(th),
		ui.Tabs("http-resp-tabs-"+id, c.respTabs).
			Selected(c.respActive).
			OnSelectItem(func(i int, _ string) { c.respActive = i }).
			TabBackground(th.Background),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(c.activeResp()).Grow(1),
	).Gap(th.Spacing.S).Padding(th.Spacing.M).Grow(1)
}

func (c *Container) activeResp() *ui.Editor {
	if c.respActive == 1 {
		return c.respHdrEd
	}
	return c.respEd
}

func (c *Container) statusLine(th *theme.Theme) ui.View {
	style := ui.Spec{}.TextColor(ui.TokenForegroundMuted)
	if c.respErr {
		style = ui.Spec{}.TextColor(ui.TokenError)
	} else if c.haveResult && c.respCode >= 200 && c.respCode < 300 {
		style = ui.Spec{}.TextColor(ui.TokenSuccess)
	}
	return ui.Text(c.statusText).Style(style)
}

func (c *Container) handleResult(r result) {
	c.pending = false
	c.haveResult = true
	if r.err != nil {
		c.respErr = true
		c.statusText = r.err.Error()
		c.respEd = replaceEditor(c.respEd, []byte(r.err.Error()))
		return
	}
	res := r.resp
	c.respErr = res.Error != nil
	c.respCode = res.StatusCode
	if c.respCode == 0 {
		c.respCode = res.StatueCode
	}
	c.respStatus = res.Status
	c.respDur = res.TimePassed
	c.respSize = len(res.Body)
	if res.Error != nil {
		c.statusText = res.Error.Error()
	} else {
		c.statusText = fmt.Sprintf("%d %s  %s  %d B", c.respCode, strings.TrimSpace(c.respStatus), c.respDur.Round(time.Millisecond), c.respSize)
	}
	body := res.Body
	if len(body) == 0 && res.JSON != "" {
		body = []byte(res.JSON)
	}
	c.respEd = replaceEditor(c.respEd, body)
	var b strings.Builder
	for k, v := range res.ResponseHeaders {
		fmt.Fprintf(&b, "%s: %s\n", k, v)
	}
	c.respHdrEd = replaceEditor(c.respHdrEd, []byte(b.String()))
}

func replaceEditor(old *ui.Editor, data []byte) *ui.Editor {
	if old != nil {
		old.Close()
	}
	return ui.NewEditor(data, highlight.NewJSON())
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
