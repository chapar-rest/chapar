// Package websocket is the editor for WebSocket requests: a connection the
// user opens, sends messages on, and watches.
package websocket

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	wsegress "github.com/chapar-rest/chapar/internal/egress/websocket"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/ui/container"
	reqicons "github.com/chapar-rest/chapar/ui/icons"
	"github.com/chapar-rest/chapar/ui/vars"
)

type connState int

const (
	stateIdle connState = iota
	stateConnecting
	stateConnected
)

// Request tabs.
const (
	tabMessage = iota
	tabSaved
	tabHeaders
	tabAuth
	tabSettings
	tabActions
	tabInfo
)

// Response tabs.
const (
	respMessages = iota
	respHeaders
	respCookies
	respTimeline
)

// connectResult is what a connect attempt delivers to the UI goroutine.
type connectResult struct {
	attempt int
	session *wsegress.Session
	resp    *egress.Response
	err     error
}

type Container struct {
	req   *domain.Request
	deps  container.Deps
	dirty bool

	msgEd   *ui.Editor
	descEd  *ui.Editor
	headers *ui.Table
	saved   *ui.Table

	varSrc    vars.Source
	preScript *ui.Editor
	authState container.AuthState

	reqTabs, respTabs     []ui.TabModel
	reqActive, respActive int

	// Connection. state, session and attempt belong to the UI goroutine;
	// results and events arrive from others.
	state     connState
	session   *wsegress.Session
	cancel    context.CancelFunc
	attempt   int
	results   chan connectResult
	sendErrs  chan error
	connected time.Time

	mu      sync.Mutex
	pending []wsegress.Event

	log       messageLog
	handshake *egress.Response
	respHdrEd *ui.Editor
	timeline  container.TimelineState
	cookies   container.CookiesState
	errText   string
}

func Open(req *domain.Request, deps container.Deps) *Container {
	r := container.CopyRequest(req)
	if r.Spec.WebSocket == nil {
		r.SetDefaultValuesForWebSocket()
	}
	w := r.Spec.WebSocket
	if w.MessageFormat == "" {
		w.MessageFormat = domain.WebSocketFormatText
	}
	c := &Container{
		req:      r,
		deps:     deps,
		results:  make(chan connectResult, 1),
		sendErrs: make(chan error, 8),
		reqTabs: []ui.TabModel{
			{Title: "Message"}, {Title: "Saved"}, {Title: "Headers"}, {Title: "Auth"},
			{Title: "Settings"}, {Title: "Actions"}, {Title: "Info"},
		},
		respTabs: []ui.TabModel{{Title: "Messages"}, {Title: "Headers"}, {Title: "Cookies"}, {Title: "Timeline"}},
	}
	c.varSrc = container.VarSource(deps, nil)
	c.msgEd = c.newMessageEditor([]byte(w.Message))
	c.descEd = container.NewDescriptionEditor(r.MetaData.Description)
	c.respHdrEd = container.NewResponseEditor(nil, highlight.Noop{})
	c.headers = container.NewKVTable("ws-hdr-"+r.MetaData.ID, c.markDirty)
	container.LoadKV(c.headers, w.Headers)
	container.AssistKV(c.headers, c.varSrc)
	c.saved = c.newSavedTable()
	c.loadSaved()
	c.authState = container.LoadAuthState(w.Auth)
	c.authState.Vars = c.varSrc
	c.authState.AllowInherit = true
	c.authState.CollectionID = r.CollectionID
	if w.PreRequest.Type == domain.PrePostTypePython && w.PreRequest.Script != "" {
		c.preScript = container.NewScriptEditor(deps, scripting.PhasePre, r.MetaData.ID+"-pre", w.PreRequest.Script)
	}
	c.log.init(r.MetaData.ID)
	return c
}

func (c *Container) ID() string           { return c.req.MetaData.ID }
func (c *Container) Kind() container.Kind { return container.KindWebSocket }
func (c *Container) Title() string        { return domain.RequestDisplayName(c.req) }
func (c *Container) Dirty() bool {
	return c.dirty || c.msgEd.Modified() || c.descEd.Modified()
}

// Close disconnects: a tab that is gone must not keep a socket open.
func (c *Container) Close() {
	c.disconnect()
	c.msgEd.Close()
	c.descEd.Close()
	c.respHdrEd.Close()
	c.log.close()
	c.timeline.Close()
	c.cookies.Close()
	if c.preScript != nil {
		c.preScript.Close()
	}
}

func (c *Container) markDirty() { c.dirty = true; c.deps.ReportDirty(true) }

func (c *Container) flush() {
	w := c.req.Spec.WebSocket
	w.Message = string(c.msgEd.Bytes())
	w.Headers = container.DumpKV(c.headers)
	c.req.MetaData.Description = string(c.descEd.Bytes())
	container.FlushAuth(&w.Auth, c.authState)
	container.FlushPreScript(&w.PreRequest, c.preScript)
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
	c.msgEd.MarkSaved()
	c.descEd.MarkSaved()
	c.deps.ReportDirty(false)
	if c.deps.Report.Saved != nil {
		c.deps.Report.Saved()
	}
	c.deps.Toast("Request saved")
	return nil
}

// Send is ⌘↵: it connects, or once connected sends the message being
// written.
func (c *Container) Send() {
	switch c.state {
	case stateIdle:
		c.connect()
	case stateConnected:
		c.sendDraft()
	}
}

func (c *Container) connect() {
	if c.state != stateIdle {
		return
	}
	c.flush()
	c.attempt++
	attempt := c.attempt
	c.state = stateConnecting
	c.errText = ""
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	req := container.CopyRequest(c.req)
	env := c.deps.ActiveEnv()
	go func() {
		session, resp, err := c.deps.Sender.ConnectWebSocket(ctx, req, env, c.onEvent)
		c.results <- connectResult{attempt: attempt, session: session, resp: resp, err: err}
		c.deps.WakeNow()
	}()
}

// disconnect closes the connection, or abandons one being opened.
func (c *Container) disconnect() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if s := c.session; s != nil {
		// Close waits for the server's answer; never on the UI goroutine.
		go s.Close()
	}
	c.session = nil
	if c.state == stateConnecting {
		c.attempt++ // the result of the abandoned attempt is dropped
	}
	c.state = stateIdle
}

// onEvent runs on the session's goroutines.
func (c *Container) onEvent(e wsegress.Event) {
	c.mu.Lock()
	c.pending = append(c.pending, e)
	c.mu.Unlock()
	c.deps.WakeNow()
}

func (c *Container) sendDraft() {
	c.sendMessage(c.req.Spec.WebSocket.MessageFormat, string(c.msgEd.Bytes()))
}

func (c *Container) sendMessage(format, body string) {
	s := c.session
	if s == nil || c.state != stateConnected {
		c.deps.Toast("Connect first")
		return
	}
	body = c.deps.Sender.ResolveText(body, c.deps.ActiveEnv())
	go func() {
		if err := s.Send(format, body); err != nil {
			c.sendErrs <- err
			c.deps.WakeNow()
		}
	}()
}

// drain applies what other goroutines delivered since the last frame.
func (c *Container) drain() {
	select {
	case r := <-c.results:
		c.handleConnect(r)
	default:
	}
	for done := false; !done; {
		select {
		case err := <-c.sendErrs:
			c.deps.Toast("Not sent: " + err.Error())
		default:
			done = true
		}
	}
	c.mu.Lock()
	events := c.pending
	c.pending = nil
	c.mu.Unlock()
	if len(events) > 0 {
		c.log.add(events)
	}
	if c.state == stateConnected && c.session != nil {
		select {
		case <-c.session.Done():
			c.session = nil
			c.cancel = nil
			c.state = stateIdle
		default:
		}
	}
}

func (c *Container) handleConnect(r connectResult) {
	if r.attempt != c.attempt {
		// The user cancelled this attempt; a late success must not linger.
		if r.session != nil {
			go r.session.Close()
		}
		return
	}
	c.cancel = nil
	c.handshake = r.resp
	if r.resp != nil {
		c.respHdrEd = container.ReplaceEditor(c.respHdrEd, []byte(handshakeHeaders(r.resp)), highlight.Noop{})
		c.cookies.Set(c.deps, r.resp)
		c.timeline.SetSteps(r.resp.Timeline)
	}
	if r.err != nil {
		c.state = stateIdle
		c.errText = r.err.Error()
		if r.resp != nil && r.resp.StatusCode != 0 {
			c.errText = fmt.Sprintf("The server answered the handshake with %d instead of switching protocols. See the Headers tab.", r.resp.StatusCode)
		}
		c.log.add([]wsegress.Event{{Dir: wsegress.DirSystem, Kind: wsegress.KindError, Info: "Could not connect: " + r.err.Error(), At: time.Now()}})
		return
	}
	c.session = r.session
	c.state = stateConnected
	c.connected = time.Now()
}

func (c *Container) Layout(ctx *ui.Ctx) ui.View {
	c.drain()
	switch c.state {
	case stateConnecting:
		ctx.Animate(30 * time.Millisecond)
	case stateConnected:
		// The status line shows how long the connection has been open.
		ctx.Animate(time.Second)
	}
	th := ctx.Theme()
	id := c.req.MetaData.ID
	w := c.req.Spec.WebSocket
	splitDir := container.SplitPaneAxis(prefs.GetGlobalConfig().Spec.General.UseHorizontalSplit)
	return ui.Column(
		ui.Row(
			container.AssistURLField(ui.TextField("ws-url-"+id, w.URL), c.varSrc).
				Placeholder("wss://…").
				OnChange(func(s string) { w.URL = s; c.markDirty() }).
				OnSubmit(func(string) {
					if c.state == stateIdle {
						c.connect()
					}
				}).
				Grow(1),
			ui.Button("ws-save-"+id, ui.Text("Save")).IconStart(icons.Save).Disabled(!c.Dirty()).OnClick(func() {
				if err := c.Save(); err != nil {
					c.deps.ShowError(err)
				}
			}),
			c.connectButton(id),
		).Gap(th.Spacing.S).Margin(th.Spacing.XS).
			MarginTop(th.Spacing.S),
		ui.Splitter("ws-split-"+id, splitDir, c.reqPane(th), c.respPane(th, ctx)).
			Percents(50, 50).
			HandleOnHover().
			Grow(1),
	).Grow(1)
}

func (c *Container) connectButton(id string) ui.View {
	switch c.state {
	case stateConnecting:
		return ui.Button("ws-connect-"+id, ui.Text("Cancel")).IconStart(icons.X).
			Tooltip("Stop connecting").OnClick(c.disconnect)
	case stateConnected:
		return ui.Button("ws-connect-"+id, ui.Text("Disconnect")).IconStart(icons.Unplug).
			OnClick(c.disconnect)
	}
	return ui.Button("ws-connect-"+id, ui.Text("Connect")).Primary().IconStart(icons.Plug).Hint("⌘↵").
		OnClick(c.connect)
}

func (c *Container) reqPane(th *theme.Theme) ui.View {
	id := c.req.MetaData.ID
	w := c.req.Spec.WebSocket
	rows := []ui.View{
		ui.Tabs("ws-req-tabs-"+id, c.reqTabs).Selected(c.reqActive).
			Closable(false).
			OnSelectItem(func(i int, _ string) { c.reqActive = i }).TabBackground(th.Background),
	}
	switch c.reqActive {
	case tabSaved:
		rows = append(rows, c.savedPane(th))
	case tabHeaders:
		rows = append(rows, container.HeadersPane(th, id, c.headers, c.req.CollectionID, c.deps.Catalog, c.markDirty))
	case tabAuth:
		rows = append(rows, container.AuthForm(th, id, &w.Auth, &c.authState, c.deps.Catalog, c.markDirty))
	case tabSettings:
		rows = append(rows, c.settingsPane(th, id, w)...)
	case tabActions:
		rows = append(rows,
			ui.Muted("Runs before every connect. A Python script sees the handshake as a GET request, and the message being written as its body."),
			container.PreRequestPane(th, c.deps, &w.PreRequest, container.PrePostOpts{ID: id + "-pre", AllowPython: true}, &c.preScript, c.markDirty),
		)
	case tabInfo:
		rows = append(rows, container.InfoPane(th, id, c.req, c.descEd, func() {
			c.markDirty()
			c.deps.ReportTitle(domain.RequestDisplayName(c.req))
		}))
	default:
		rows = append(rows, c.messagePane(th)...)
	}
	return ui.Column(
		ui.Column(rows...).
			Padding(th.Spacing.S).
			Gap(th.Spacing.S).Grow(1),
	)
}

var formatOptions = []ui.SelectOption{
	{Label: "Text", Value: domain.WebSocketFormatText},
	{Label: "JSON", Value: domain.WebSocketFormatJSON},
	{Label: "Binary (base64)", Value: domain.WebSocketFormatBinary},
}

func (c *Container) messagePane(th *theme.Theme) []ui.View {
	id := c.req.MetaData.ID
	w := c.req.Spec.WebSocket
	editor := ui.View(ui.ViewOf(c.msgEd).Grow(1))
	if w.MessageFormat == domain.WebSocketFormatJSON {
		editor = container.JSONBodyEditor("ws-msg-"+id, c.msgEd, c.deps, nil)
	}
	return []ui.View{
		ui.Row(
			ui.Select("ws-format-"+id, formatOptions).Width(170).
				Selected(optionIndex(w.MessageFormat, formatOptions)).
				OnChange(func(v string) { c.setFormat(v) }),
			ui.Spacer(),
			ui.Button("ws-keep-"+id, ui.Text("Save message")).IconStart(icons.BookmarkPlus).
				Tooltip("Keep this message in the Saved tab").
				OnClick(c.saveDraft),
			ui.Button("ws-send-"+id, ui.Text("Send")).Primary().IconStart(icons.SendHorizontal).Hint("⌘↵").
				Disabled(c.state != stateConnected).
				OnClick(c.sendDraft),
		).Gap(th.Spacing.S).Align(ui.AlignCenter),
		editor,
	}
}

func (c *Container) setFormat(format string) {
	w := c.req.Spec.WebSocket
	if w.MessageFormat == format {
		return
	}
	w.MessageFormat = format
	data := c.msgEd.Bytes()
	c.msgEd.Close()
	c.msgEd = c.newMessageEditor(data)
	// The new editor starts unmodified; the dirty flag keeps the edits.
	c.markDirty()
}

func (c *Container) newMessageEditor(data []byte) *ui.Editor {
	bodyType := domain.RequestBodyTypeText
	if c.req.Spec.WebSocket.MessageFormat == domain.WebSocketFormatJSON {
		bodyType = domain.RequestBodyTypeJSON
	}
	return container.NewBodyEditor(c.deps, "ws-msg-"+c.req.MetaData.ID, bodyType, data)
}

func (c *Container) settingsPane(th *theme.Theme, id string, w *domain.WebSocketRequestSpec) []ui.View {
	set := &w.Settings
	protocols := ui.FormText("ws-protocols-"+id, "Subprotocols", "Offered in Sec-WebSocket-Protocol, comma separated, most preferred first", strings.Join(w.Subprotocols, ", "), func(v string) {
		w.Subprotocols = splitList(v)
		c.markDirty()
	})
	protocols.Optional = true
	protocols.Placeholder = "None"
	items := []ui.FormItem{
		protocols,
		ui.FormNumber("ws-timeout-"+id, "Connect timeout (s)", "How long the handshake may take; zero uses the request timeout from Settings", float64(set.ConnectTimeoutSec), 0, 3600, 1, func(v float64) {
			set.ConnectTimeoutSec = int(v)
			c.markDirty()
		}),
		ui.FormNumber("ws-ping-"+id, "Ping interval (s)", "Send a ping this often to keep the connection alive; zero never pings", float64(set.PingIntervalSec), 0, 3600, 1, func(v float64) {
			set.PingIntervalSec = int(v)
			c.markDirty()
		}),
		ui.FormSwitch("ws-deflate-"+id, "Compression", "Offer permessage-deflate", set.Compression, func(v bool) {
			set.Compression = v
			c.markDirty()
		}),
	}
	return []ui.View{
		ui.Scroll("ws-settings-scroll-"+id, ui.Column(
			ui.Form("ws-settings-"+id, items...),
			ui.Muted("Changes apply the next time you connect."),
		).Gap(th.Spacing.S)).Grow(1),
	}
}

func (c *Container) respPane(th *theme.Theme, ctx *ui.Ctx) ui.View {
	id := c.req.MetaData.ID

	var content ui.View
	switch c.respActive {
	case respHeaders:
		if c.handshake == nil {
			content = ui.Muted("Connect to see the handshake headers.")
		} else {
			content = container.ResponseEditorMenu(c.respHdrEd, ctx, c.deps, "handshake.txt")
		}
	case respCookies:
		content = container.CookiesView("ws-ck-"+id, th, ctx, c.deps, &c.cookies, false)
	case respTimeline:
		var steps []egress.TimelineStep
		if c.handshake != nil {
			steps = c.handshake.Timeline
		}
		content = container.TimelineView("ws-tl-"+id, th, ctx, c.deps, &c.timeline, steps)
	default:
		if c.errText != "" && c.log.len() <= 1 {
			content = container.ErrorView("ws-err-"+id, th, ctx, c.deps, c.errText)
			break
		}
		content = c.log.view(th, ctx, c.deps)
	}

	return ui.Column(
		ui.Column(
			ui.Text(c.statusText()).Style(c.statusStyle()),
			ui.Tabs("ws-resp-tabs-"+id, c.respTabs).Selected(c.respActive).
				Closable(false).
				OnSelectItem(func(i int, _ string) { c.respActive = i }),
			content,
		).Radius(th.Radius.Medium).
			Border(ui.TokenBorder, th.Stroke.Thick).Margin(th.Spacing.XS).
			BorderStyle(ui.BorderDotted).
			Padding(th.Spacing.S).
			Gap(th.Spacing.S).Grow(1),
	)
}

func (c *Container) statusText() string {
	switch c.state {
	case stateConnecting:
		return "Connecting…"
	case stateConnected:
		up := time.Since(c.connected).Truncate(time.Second)
		sent, received := c.log.counts()
		return fmt.Sprintf("Connected  %s  ↑ %d  ↓ %d", up, sent, received)
	}
	if c.errText != "" {
		return "Connection failed"
	}
	if c.handshake != nil {
		return "Disconnected"
	}
	return "Not connected"
}

func (c *Container) statusStyle() ui.Spec {
	switch {
	case c.state == stateConnected:
		return ui.Spec{}.TextColor(ui.TokenSuccess)
	case c.state == stateIdle && c.errText != "":
		return ui.Spec{}.TextColor(ui.TokenError)
	}
	return ui.Spec{}.TextColor(ui.TokenForegroundMuted)
}

// TabIcon shows the request's badge, the one its row in the tree shows.
func (c *Container) TabIcon(th *theme.Theme) (icons.Icon, render.Color) {
	return reqicons.Badge(c.req), reqicons.Color(c.req, th)
}

func handshakeHeaders(res *egress.Response) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# --- Request Headers ---\n")
	for _, k := range sortedKeys(res.RequestHeaders) {
		fmt.Fprintf(&b, "%s: %s\n", k, res.RequestHeaders[k])
	}
	fmt.Fprintf(&b, "\n# --- Response Headers (%d) ---\n", res.StatusCode)
	for _, k := range sortedKeys(res.ResponseHeaders) {
		fmt.Fprintf(&b, "%s: %s\n", k, res.ResponseHeaders[k])
	}
	return b.String()
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
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

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
