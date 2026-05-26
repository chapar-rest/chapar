package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
)

type GeneralRequestSettings struct {
	HTTPVersion            string
	RequestTimeoutSec      int
	ResponseSizeMb         int
	FollowRedirects        bool
	VaidateTLSCertificates bool
}

type GeneralHeaderSettings struct {
	SendNoCacheHeader     bool
	SendChaparAgentHeader bool
}

type GeneralUISettings struct {
	UseHorizontalSplit bool
}

// GeneralSettings configures HTTP and UI behavior.
type GeneralSettings struct {
	Request GeneralRequestSettings
	Headers GeneralHeaderSettings
	UI      GeneralUISettings
}

func (g *GeneralSettings) Defaults() {}

func (g *GeneralSettings) FromConfig(cfg domain.GeneralConfig) {
	g.Request.HTTPVersion = cfg.HTTPVersion
	g.Request.RequestTimeoutSec = cfg.RequestTimeoutSec
	g.Request.ResponseSizeMb = cfg.ResponseSizeMb
	g.Request.FollowRedirects = cfg.FollowRedirects
	g.Request.VaidateTLSCertificates = cfg.VaidateTLSCertificates
	g.Headers.SendNoCacheHeader = cfg.SendNoCacheHeader
	g.Headers.SendChaparAgentHeader = cfg.SendChaparAgentHeader
	g.UI.UseHorizontalSplit = cfg.UseHorizontalSplit
}

func (g *GeneralSettings) ToConfig(cfg *domain.GeneralConfig) {
	cfg.HTTPVersion = g.Request.HTTPVersion
	cfg.RequestTimeoutSec = g.Request.RequestTimeoutSec
	cfg.ResponseSizeMb = g.Request.ResponseSizeMb
	cfg.FollowRedirects = g.Request.FollowRedirects
	cfg.VaidateTLSCertificates = g.Request.VaidateTLSCertificates
	cfg.SendNoCacheHeader = g.Headers.SendNoCacheHeader
	cfg.SendChaparAgentHeader = g.Headers.SendChaparAgentHeader
	cfg.UseHorizontalSplit = g.UI.UseHorizontalSplit
}
