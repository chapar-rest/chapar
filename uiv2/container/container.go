package container

import (
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/uiv2/sender"
	"github.com/mirzakhany/yoga/ui"
)

type Kind string

const (
	KindHTTP       Kind = "request-http"
	KindGRPC       Kind = "request-grpc"
	KindGraphQL    Kind = "request-graphql"
	KindCollection Kind = "collection"
	KindEnv        Kind = "environment"
)

// Container is the only editor contract. Protocol-specific UI stays inside each implementation.
type Container interface {
	ID() string
	Kind() Kind
	Title() string
	Dirty() bool
	Layout(c *ui.Ctx) ui.View
	Close()
	Save() error
	Send()
}

// Reporter is how a container tells the workspace about its own state.
type Reporter struct {
	Dirty   func(dirty bool)
	Title   func(title string)
	Saved   func()
	Error   func(error)
	Toast   func(msg string)
	Refresh func()
}

// Deps is everything a container needs to be self-service.
// Dialogs/Files/Toasts are window services; prefer Report callbacks filled by the app.
type Deps struct {
	Repo      repository.RepositoryV2
	Catalog   Catalog
	Sender    *sender.Service
	Dialogs   func() *ui.DialogHost
	Files     func() *ui.FileDialog
	Toasts    func() *ui.ToastHost
	Wake      func()
	ActiveEnv func() *domain.Environment
	Report    Reporter
}

// Catalog is the subset of app catalog a container may query.
type Catalog interface {
	RequestByID(id string) *domain.Request
	CollectionByID(id string) *domain.Collection
	EnvironmentByID(id string) *domain.Environment
	ReplaceEnvironment(env *domain.Environment)
	Load() error
	AllCollections() []*domain.Collection
	StandaloneRequests() []*domain.Request
	ProtoFileList() []*domain.ProtoFile
}

// OpenSpec is the factory input. Kind is inferred from the non-nil document.
type OpenSpec struct {
	Request    *domain.Request
	Collection *domain.Collection
	Env        *domain.Environment
	Deps       Deps
}

func (d Deps) ReportDirty(dirty bool) {
	if d.Report.Dirty != nil {
		d.Report.Dirty(dirty)
	}
}

func (d Deps) ReportTitle(title string) {
	if d.Report.Title != nil {
		d.Report.Title(title)
	}
}

func (d Deps) ShowError(err error) {
	if err == nil {
		return
	}
	if d.Report.Error != nil {
		d.Report.Error(err)
		return
	}
	if d.Dialogs != nil {
		if host := d.Dialogs(); host != nil {
			host.ShowError("Error", err.Error(), nil)
		}
	}
}

func (d Deps) WakeNow() {
	if d.Wake != nil {
		d.Wake()
	}
}

func (d Deps) Toast(msg string) {
	if d.Report.Toast != nil {
		d.Report.Toast(msg)
		return
	}
	if d.Toasts != nil {
		if host := d.Toasts(); host != nil {
			host.Show(msg, ui.ToastInfo, 3*time.Second)
		}
	}
}
