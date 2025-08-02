package app

import (
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/scripting"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/fonts"
	"github.com/chapar-rest/chapar/ui/navigator"
)

type Base struct {
	// Window is the main application window.
	Theme      *chapartheme.Theme
	Window     *app.Window
	Navigator  *navigator.Navigator
	Repository repository.RepositoryV2
	Explorer   *explorer.Explorer
	// Modal is the modal layer for displaying dialogs and modals.
	// its in the base because it is used in many places and we want all views to have access to it.
	Modal *component.ModalLayer

	// TODO state should eventually be removed component should use repo directly.
	ProtoFilesState   *state.ProtoFiles
	RequestsState     *state.Requests
	EnvironmentsState *state.Environments
	WorkspacesState   *state.Workspaces

	// services
	GrpcService   *grpc.Service
	RestService   *rest.Service
	EgressService *egress.Service

	// scripting executor
	Executor scripting.Executor
}

func NewBase(appVersion string, w *app.Window, navi *navigator.Navigator) (*Base, error) {
	fontCollection, err := fonts.Prepare()
	if err != nil {
		return nil, err
	}

	appState := prefs.GetAppState()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(fontCollection))
	th := chapartheme.New(theme, appState.Spec.DarkMode)
	explorerController := explorer.NewExplorer(w)

	// create file storage in user's home directory
	repo, err := repository.NewFilesystemV2(prefs.GetWorkspacePath(), appState.Spec.ActiveWorkspace.Name)
	if err != nil {
		return nil, err
	}

	// init state
	protoFilesState := state.NewProtoFiles(repo)
	requestsState := state.NewRequests(repo)
	environmentsState := state.NewEnvironments(repo)
	workspacesState, err := state.NewWorkspaces(repo)
	if err != nil {
		return nil, err
	}

	// init services
	grpcService := grpc.NewService(appVersion, requestsState, environmentsState, protoFilesState)
	restService := rest.New(requestsState, environmentsState, appVersion)
	egressService := egress.New(requestsState, environmentsState, restService, grpcService, nil)

	modal := component.NewModal()

	return &Base{
		Theme:             th,
		Window:            w,
		Navigator:         navi,
		Repository:        repo,
		Modal:             modal,
		Explorer:          explorerController,
		ProtoFilesState:   protoFilesState,
		RequestsState:     requestsState,
		EnvironmentsState: environmentsState,
		WorkspacesState:   workspacesState,
		GrpcService:       grpcService,
		RestService:       restService,
		EgressService:     egressService,
		Executor:          nil, // scripting executor will be set later,
	}, nil
}

func (b *Base) SetModal(mw func(gtx layout.Context) layout.Dimensions) {
	// this hack is needed to avoid scrim being disparaged when user is clicking on it
	b.Modal.Scrim.Clickable = widget.Clickable{}
	b.Modal.VisibilityAnimation = component.VisibilityAnimation{
		Duration: 200 * time.Millisecond,
		State:    component.Invisible,
	}

	b.Modal.Widget = func(gtx layout.Context, theme *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions {
		return mw(gtx)
	}

	b.Modal.VisibilityAnimation.Appear(time.Now())
}
