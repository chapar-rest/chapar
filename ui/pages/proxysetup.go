package pages

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"

	"github.com/chapar-rest/chapar/internal/mitm"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/ui/container"
	"github.com/chapar-rest/chapar/ui/langsrv"
)

// proxySetup is the readiness checklist: certificate, trust, proxy,
// clients and other devices, each with what to do when it isn't ready.
type proxySetup struct {
	p *Proxy

	mu       sync.Mutex
	checking bool
	checked  bool
	trusted  bool
	sys      mitm.SystemProxy
	sysErr   error
	busy     bool // a trust change is waiting on the OS prompt
}

func newProxySetup(p *Proxy) *proxySetup {
	s := &proxySetup{p: p}
	s.recheck()
	return s
}

// recheck refreshes the trust and system proxy state in the background:
// the platform verifier and scutil each take a moment.
func (s *proxySetup) recheck() {
	s.mu.Lock()
	if s.checking {
		s.mu.Unlock()
		return
	}
	s.checking = true
	s.mu.Unlock()
	go func() {
		trusted := false
		if ca, err := s.p.svc().CA(); err == nil {
			trusted, _ = ca.TrustedByOS()
		}
		sys, sysErr := mitm.ReadSystemProxy()
		s.mu.Lock()
		s.checking, s.checked = false, true
		s.trusted, s.sys, s.sysErr = trusted, sys, sysErr
		s.mu.Unlock()
		s.p.wake()
	}()
}

type setupState struct {
	checked, trusted, busy bool
	sys                    mitm.SystemProxy
	sysErr                 error
}

func (s *proxySetup) state() setupState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return setupState{s.checked, s.trusted, s.busy, s.sys, s.sysErr}
}

// ready reports whether HTTPS capture works on this machine: a CA exists,
// the OS trusts it and the proxy is running.
func (s *proxySetup) ready() bool {
	_, err := s.p.svc().CA()
	return err == nil && s.state().trusted && s.p.svc().Running()
}

func (s *proxySetup) button(c *ui.Ctx) ui.View {
	b := ui.Button("proxy-setup", ui.Text("Setup")).Ghost().HoverFill().OnClick(func() { s.show(c) })
	if s.ready() {
		return b.IconStart(icons.ShieldCheck).Tooltip("Ready to capture HTTPS")
	}
	return b.IconStart(icons.ShieldAlert).Tooltip("Some setup steps are left")
}

func (s *proxySetup) show(c *ui.Ctx) {
	s.recheck()
	c.Dialogs().Show(ui.DialogOpts{
		Title:   "Proxy setup",
		Width:   680,
		Height:  600,
		Body:    s.body,
		Actions: []ui.DialogAction{{Label: "Done", Primary: true}},
	})
}

type stepStatus int

const (
	stepOK stepStatus = iota
	stepTodo
	stepInfo
)

func (s *proxySetup) body(c *ui.Ctx) ui.View {
	th := c.Theme()
	st := s.state()
	svc := s.p.svc()
	ca, caErr := svc.CA()
	port := prefs.GetGlobalConfig().Spec.Proxy.ListenPort()

	var summary ui.View
	switch {
	case !st.checked:
		summary = ui.Row(ui.Spinner("proxy-setup-spin", 16), ui.Text("Checking…")).Gap(th.Spacing.S).Align(ui.AlignCenter)
	case s.ready():
		summary = ui.Badge("Ready to capture HTTPS").Tone(ui.BadgeSuccess)
	default:
		summary = ui.Badge("Finish the steps below to capture HTTPS").Tone(ui.BadgeWarning)
	}

	steps := []ui.View{
		ui.Row(summary, ui.Spacer(),
			ui.Button("proxy-setup-recheck", ui.Text("Check again")).IconStart(icons.RefreshCw).Ghost().HoverFill().
				OnClick(s.recheck),
		).Align(ui.AlignCenter),
		s.certStep(c, ca, caErr),
		s.trustStep(c, ca, st),
		s.runStep(c),
		s.clientStep(c, ca, st, port),
		s.devicesStep(c, port),
	}
	return ui.Scroll("proxy-setup-scroll",
		ui.Column(steps...).Gap(th.Spacing.M).Padding(th.Spacing.XS),
	).Grow(1)
}

func (s *proxySetup) certStep(c *ui.Ctx, ca *mitm.CA, caErr error) ui.View {
	th := c.Theme()
	if caErr != nil {
		return stepCard(th, stepTodo, "1. Create Chapar's certificate",
			"Chapar signs a certificate for each site it decrypts. It needs its own certificate authority for that, made on this computer and never shared.",
			ui.Row(ui.Button("proxy-ca-create", ui.Text("Create certificate")).Primary().IconStart(icons.ShieldCheck).
				OnClick(func() {
					if _, err := s.p.svc().EnsureCA(); err != nil {
						s.p.deps.Error(err)
					}
					s.recheck()
				})),
		)
	}
	details := ui.Column(
		kv(th, "Name", ca.Name()),
		kv(th, "SHA-256", ca.Fingerprint()),
		kv(th, "Expires", ca.NotAfter().Format("2 Jan 2006")),
		kv(th, "File", ca.CertPath()),
	).Gap(th.Spacing.XXS)
	return stepCard(th, stepOK, "1. Chapar's certificate", "", details,
		ui.Row(
			ui.Button("proxy-ca-reveal", ui.Text("Show file")).IconStart(icons.FolderOpen).OnClick(func() {
				if err := langsrv.OpenURL(filepath.Dir(ca.CertPath())); err != nil {
					s.p.deps.Error(err)
				}
			}),
			ui.Button("proxy-ca-copy", ui.Text("Copy path")).IconStart(icons.Copy).OnClick(func() {
				s.copy(c, ca.CertPath())
			}),
			ui.Spacer(),
			ui.Button("proxy-ca-regen", ui.Text("Replace…")).IconStart(icons.RefreshCw).Ghost().HoverFill().OnClick(func() {
				s.confirmRegenerate(c)
			}),
		).Gap(th.Spacing.S),
	)
}

func (s *proxySetup) confirmRegenerate(c *ui.Ctx) {
	c.Dialogs().ShowAction("Replace the certificate?",
		"Chapar makes a new certificate authority. Every computer and phone that trusts the current one must trust the new one before HTTPS capture works there again.",
		func() {
			old, _ := s.p.svc().CA()
			if old != nil && mitm.CanInstallTrust() {
				// Best effort: don't leave a trusted root nobody uses.
				_ = mitm.RemoveTrust(context.Background(), old)
			}
			if _, err := s.p.svc().RegenerateCA(); err != nil {
				s.p.deps.Error(err)
			}
			s.recheck()
		}, nil)
}

func (s *proxySetup) trustStep(c *ui.Ctx, ca *mitm.CA, st setupState) ui.View {
	th := c.Theme()
	title := "2. Trust it on this computer"
	if ca == nil {
		return stepCard(th, stepTodo, title, "Create the certificate first.")
	}
	if st.trusted {
		var actions ui.View
		if mitm.CanInstallTrust() {
			actions = ui.Row(ui.Button("proxy-trust-remove", ui.Text("Remove trust")).Ghost().HoverFill().
				Disabled(st.busy).OnClick(func() { s.changeTrust(ca, false) }))
		}
		return stepCard(th, stepOK, title, "This computer trusts Chapar's certificate, so apps here accept the sites Chapar decrypts.", actions)
	}

	var children []ui.View
	if mitm.CanInstallTrust() {
		label := "Trust certificate"
		if st.busy {
			label = "Waiting for your password…"
		}
		children = append(children,
			ui.Row(ui.Button("proxy-trust-add", ui.Text(label)).Primary().IconStart(icons.ShieldCheck).
				Disabled(st.busy).OnClick(func() { s.changeTrust(ca, true) })),
			container.MutedParagraph("macOS asks for your password to add it to your login keychain. Or do it by hand:"),
		)
	}
	for i, step := range mitm.TrustSteps(ca) {
		children = append(children, s.manualStep(c, "proxy-trust-step-"+strconv.Itoa(i), step))
	}
	return stepCard(th, stepTodo, title,
		"Until this computer trusts Chapar's certificate, apps here refuse HTTPS sites going through the proxy.",
		children...)
}

func (s *proxySetup) changeTrust(ca *mitm.CA, trust bool) {
	s.mu.Lock()
	s.busy = true
	s.mu.Unlock()
	go func() {
		var err error
		if trust {
			err = mitm.InstallTrust(context.Background(), ca)
		} else {
			err = mitm.RemoveTrust(context.Background(), ca)
		}
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			s.p.deps.Error(fmt.Errorf("couldn't change the certificate's trust: %w", err))
		}
		s.recheck()
	}()
}

func (s *proxySetup) runStep(c *ui.Ctx) ui.View {
	th := c.Theme()
	svc := s.p.svc()
	if svc.Running() {
		return stepCard(th, stepOK, "3. Start the proxy", "Listening on "+strings.Join(svc.ClientAddrs(), ", ")+".")
	}
	detail := "The proxy is off."
	if err := svc.LastError(); err != nil {
		detail = "The proxy stopped: " + err.Error()
	}
	return stepCard(th, stepTodo, "3. Start the proxy", detail,
		ui.Row(ui.Button("proxy-setup-start", ui.Text("Start proxy")).Primary().IconStart(icons.Play).OnClick(s.p.Start)))
}

func (s *proxySetup) clientStep(c *ui.Ctx, ca *mitm.CA, st setupState, port int) ui.View {
	th := c.Theme()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	title := "4. Send traffic through Chapar"
	if st.sysErr == nil && st.sys.PointsTo(port) {
		return stepCard(th, stepOK, title, "This computer's proxy setting points to Chapar, so browsers and most apps send their traffic through it.")
	}
	children := []ui.View{}
	if runtime.GOOS == "darwin" {
		children = append(children,
			container.MutedParagraph("For browsers and apps: in System Settings › Network, open your network's Details › Proxies and set both the web (HTTP) and secure web (HTTPS) proxy to "+addr+". Turn them off again when you're done."),
			ui.Row(ui.Button("proxy-open-network", ui.Text("Open Network settings")).IconStart(icons.ExternalLink).OnClick(func() {
				_ = langsrv.OpenURL("x-apple.systempreferences:com.apple.Network-Settings.extension")
			})),
		)
	} else {
		children = append(children,
			container.MutedParagraph("For browsers and apps: set this computer's HTTP and HTTPS proxy to "+addr+" in its network settings, or only in the app you want to inspect."))
	}
	if ca != nil {
		children = append(children,
			container.MutedParagraph("For terminal tools, run this in the shell first. It also makes curl, Node and Python trust the certificate:"),
			s.manualStep(c, "proxy-env", mitm.TrustStep{Command: mitm.ClientEnv(addr, ca)}),
		)
	}
	return stepCard(th, stepInfo, title, "Chapar never changes your network settings itself.", children...)
}

func (s *proxySetup) devicesStep(c *ui.Ctx, port int) ui.View {
	th := c.Theme()
	title := "5. Phones and other devices (optional)"
	cfg := prefs.GetGlobalConfig().Spec.Proxy
	if !cfg.AllowLAN {
		return stepCard(th, stepInfo, title,
			"To capture a phone's traffic, allow devices on your local network in Settings › Proxy, then restart the proxy.",
			ui.Row(ui.Button("proxy-open-settings", ui.Text("Open settings")).IconStart(icons.Settings).OnClick(func() {
				c.Dialogs().Close()
				if s.p.deps.OpenSettings != nil {
					s.p.deps.OpenSettings()
				}
			})))
	}
	addrs := mitm.LANAddrs()
	if len(addrs) == 0 {
		return stepCard(th, stepTodo, title, "This computer isn't on a network other devices can reach.")
	}
	rows := []ui.View{
		container.MutedParagraph("On the device, set the Wi-Fi network's HTTP proxy to manual with this server and port:"),
	}
	for _, a := range addrs {
		rows = append(rows, kv(th, "Server", a), kv(th, "Port", strconv.Itoa(port)))
	}
	rows = append(rows,
		container.MutedParagraph("Then open http://"+mitm.CertSiteHost+" in the device's browser. It walks through installing and trusting the certificate."),
	)
	return stepCard(th, stepInfo, title, "", rows...)
}

// manualStep shows one hand-run step, with a copy button for its command.
func (s *proxySetup) manualStep(c *ui.Ctx, id string, step mitm.TrustStep) ui.View {
	th := c.Theme()
	var views []ui.View
	if step.Text != "" {
		views = append(views, ui.Paragraph(step.Text))
	}
	if step.Command != "" {
		cmd := step.Command
		views = append(views, ui.Row(
			ui.Paragraph(cmd).Grow(1),
			ui.IconButton(id+"-copy", icons.Copy).Tooltip("Copy").OnClick(func() { s.copy(c, cmd) }),
		).Gap(th.Spacing.S).Align(ui.AlignStart).
			Padding(th.Spacing.S).Radius(th.Radius.Small).Background(ui.TokenChromeMuted))
	}
	return ui.Column(views...).Gap(th.Spacing.XS)
}

func (s *proxySetup) copy(c *ui.Ctx, text string) {
	if clip := c.Clipboard(); clip != nil {
		clip.Set(text)
		s.p.deps.Toast("Copied")
	}
}

func stepCard(th *theme.Theme, status stepStatus, title, detail string, children ...ui.View) ui.View {
	var icon icons.Icon
	var color render.Color
	switch status {
	case stepOK:
		icon, color = icons.CircleCheck, th.Success
	case stepTodo:
		icon, color = icons.TriangleAlert, th.Warning
	default:
		icon, color = icons.Info, th.ForegroundMuted
	}
	body := []ui.View{ui.Strong(title)}
	if detail != "" {
		body = append(body, container.MutedParagraph(detail))
	}
	for _, ch := range children {
		if ch != nil {
			body = append(body, ch)
		}
	}
	return ui.Row(
		ui.Icon(icon, 18, color),
		ui.Column(body...).Gap(th.Spacing.S).Grow(1),
	).Gap(th.Spacing.M).Align(ui.AlignStart).
		Padding(th.Spacing.M).
		Radius(th.Radius.Medium).
		Border(ui.TokenBorder, th.Stroke.Thin)
}

func kv(th *theme.Theme, k, v string) ui.View {
	return ui.Row(
		ui.Caption(k).Width(70),
		ui.Paragraph(v).Grow(1),
	).Gap(th.Spacing.S).Align(ui.AlignStart)
}
