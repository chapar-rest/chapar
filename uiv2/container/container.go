package container

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
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
}

// Reporter is how a container tells the workspace about its own state.
type Reporter struct {
	Dirty   func(dirty bool)
	Title   func(title string)
	Saved   func()
	Refresh func()
}

// Deps is everything a container needs to be self-service.
type Deps struct {
	Repo      repository.RepositoryV2
	Catalog   Catalog
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
