package requests

import (
	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	giox "gioui.org/x/component"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/ui"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/navigator"
	"github.com/chapar-rest/chapar/ui/pages/requests/graphql"
	"github.com/chapar-rest/chapar/ui/pages/requests/grpc"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/pages/requests/collections"
	"github.com/chapar-rest/chapar/ui/pages/requests/restful"
	"github.com/chapar-rest/chapar/ui/pages/tips"
	"github.com/chapar-rest/chapar/ui/widgets"
)

var (
	_ navigator.View = &View{}

	newGRPCRequest    = modals.NewCreateItem(string(domain.RequestTypeGRPC), widgets.GRPCIcon, "GRPC")
	newHTTPRequest    = modals.NewCreateItem(string(domain.RequestTypeHTTP), widgets.HTTPIcon, "HTTP")
	newGraphQLRequest = modals.NewCreateItem(string(domain.RequestTypeGraphQL), widgets.GraphQLIcon, "GraphQL")
	newHCollection    = modals.NewCreateItem(string(domain.KindCollection), widgets.CollectionIcon, "Collection")
)

const (
	MenuDuplicate         = "Duplicate"
	MenuDelete            = "Delete"
	MenuAddHTTPRequest    = "Add HTTP Request"
	MenuAddGRPCRequest    = "Add GRPC Request"
	MenuAddGraphQLRequest = "Add GraphQL Request"
	MenuView              = "View"
)

// RequestController is the interface the View uses to notify its controller of events.
type RequestController interface {
	OnNewRequest(requestType domain.RequestType)
	OnImport(importType string)
	OnNewCollection()
	OnTitleChanged(id, title, containerType string)
	OnTreeViewNodeClicked(id string)
	OnTreeViewMenuClicked(id, action string)
	OnTabClose(id string)
	OnDataChanged(id string, data any, containerType string)
	OnSave(id string)
	OnSubmit(id, containerType string)
	OnCopyResponse(gtx layout.Context, dataType, data string)
	OnPostRequestSetChanged(id string, statusCode int, item, from, fromKey string)
	OnSetOnTriggerRequestChanged(id, collectionID, requestID string)
	OnBinaryFileSelect(id string)
	OnFormDataFileSelect(requestID, fieldID string)
	OnServerInfoReload(id string)
	OnGrpcInvoke(id string)
	OnGrpcLoadRequestExample(id string)
	OnRequestTabChanged(id, tab string)
	OnCreateCollectionFromMethods(requestID string)
}

type View struct {
	theme  *chapartheme.Theme
	window *app.Window

	*ui.Base

	// add menu
	newRequestButton widget.Clickable
	importButton     widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	split     widgets.SplitView
	tabHeader *widgets.Tabs

	controller RequestController

	// state
	containers    *safemap.Map[Container]
	openTabs      *safemap.Map[*widgets.Tab]
	treeViewNodes *safemap.Map[*widgets.TreeNode]

	tipsView *tips.Tips

	explorer *explorer.Explorer
}

func (v *View) OnEnter() {
}

func (v *View) Info() navigator.Info {
	return navigator.Info{
		ID:    "requests",
		Title: "Requests",
		Icon:  widgets.SwapHoriz,
	}
}

func NewView(b *ui.Base) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	v := &View{
		Base:              b,
		window:            b.Window,
		theme:             b.Theme,
		treeViewSearchBox: search,
		tabHeader:         widgets.NewTabs([]*widgets.Tab{}, nil),
		treeView:          widgets.NewTreeView([]*widgets.TreeNode{}),
		split: widgets.SplitView{
			// Ratio:       -0.64,
			Resize: giox.Resize{
				Ratio: 0.19,
			},
			BarWidth: unit.Dp(2),
		},
		containers:    safemap.New[Container](),
		treeViewNodes: safemap.New[*widgets.TreeNode](),
		openTabs:      safemap.New[*widgets.Tab](),
		tipsView:      tips.New(),
		explorer:      b.Explorer,
	}

	v.tabHeader.SetMaxTitleWidth(20)

	return v
}

func (v *View) SetController(c RequestController) {
	v.controller = c
	v.treeView.OnNodeDoubleClick(func(node *widgets.TreeNode) {
		if v.controller != nil {
			v.controller.OnTreeViewNodeClicked(node.Identifier)
		}
	})
	v.treeView.OnNodeClick(func(node *widgets.TreeNode) {
		if v.controller != nil {
			v.controller.OnTreeViewNodeClicked(node.Identifier)
		}
	})
	v.treeView.SetOnMenuItemClick(func(node *widgets.TreeNode, item string) {
		if v.controller != nil {
			v.controller.OnTreeViewMenuClicked(node.Identifier, item)
		}
	})
}

func (v *View) SetPreRequestCollections(id string, collections []*domain.Collection, selectedID string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetPreRequestCollections(collections, selectedID)
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetPreRequestCollections(collections, selectedID)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetPreRequestCollections(collections, selectedID)
		}
	}
}

func (v *View) SetPreRequestRequests(id string, requests []*domain.Request, selectedID string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetPreRequestRequests(requests, selectedID)
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetPreRequestRequests(requests, selectedID)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetPreRequestRequests(requests, selectedID)
		}
	}
}

func (v *View) showError(err error) {
	m := modals.NewError(err)
	v.Base.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.OKBtn.Clicked(gtx) {
			v.Base.CloseModal()
		}
		return m.Layout(gtx, v.Theme)
	})
}

func (v *View) SetPostRequestSetValues(id string, set domain.PostRequestSet) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetPostRequestSetValues(set)
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetPostRequestSetValues(set)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetPostRequestSetValues(set)
		}
	}
}

func (v *View) SetPostRequestSetPreview(id, preview string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetPostRequestSetPreview(preview)
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetPostRequestSetPreview(preview)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetPostRequestSetPreview(preview)
		}
	}
}

func (v *View) AddRequestTreeViewNode(req *domain.Request) {
	node := &widgets.TreeNode{
		Text:        req.MetaData.Name,
		Identifier:  req.MetaData.ID,
		MenuOptions: []string{MenuView, MenuDuplicate, MenuDelete},
		Meta:        safemap.New[string](),
	}

	setNodePrefix(req, node)

	node.Meta.Set(TypeMeta, TypeRequest)
	v.treeView.AddNode(node)
	v.treeViewNodes.Set(req.MetaData.ID, node)
}

func (v *View) AddCollectionTreeViewNode(collection *domain.Collection) {
	node := &widgets.TreeNode{
		Text:        collection.MetaData.Name,
		Identifier:  collection.MetaData.ID,
		Children:    make([]*widgets.TreeNode, 0),
		MenuOptions: []string{MenuAddHTTPRequest, MenuAddGRPCRequest, MenuAddGraphQLRequest, MenuDuplicate, MenuView, MenuDelete},
		Meta:        safemap.New[string](),
	}

	node.Meta.Set(TypeMeta, TypeCollection)
	v.treeView.AddNode(node)
	v.treeViewNodes.Set(collection.MetaData.ID, node)
}

func (v *View) RemoveTreeViewNode(id string) {
	if _, ok := v.treeViewNodes.Get(id); !ok {
		return
	}

	v.treeView.RemoveNode(id)
	v.treeViewNodes.Delete(id)
}

func (v *View) SetSetGrpcRequestBody(id, body string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetRequestBody(body)
		}
	}
}

func (v *View) SetBinaryBodyFilePath(id, filePath string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetBinaryBodyFilePath(filePath)
		}
	}
}

func (v *View) SetGRPCServices(id string, services []domain.GRPCService) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetServices(services)
		}
	}
}

func (v *View) SetGRPCMethodsLoading(id string, loading bool) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetMethodsLoading(loading)
		}
	}
}

func (v *View) AddFileToFormData(requestId, fieldId, filePath string) {
	if ct, ok := v.containers.Get(requestId); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.AddFileToFormData(fieldId, filePath)
		}
	}
}

func (v *View) ExpandTreeViewNode(id string) {
	node, ok := v.treeViewNodes.Get(id)
	if !ok {
		return
	}

	v.treeView.ExpandNode(node.Identifier)
}

func (v *View) UpdateTabTitle(id, title string) {
	if tab, ok := v.openTabs.Get(id); ok {
		tab.Title = title
	}
}
func (v *View) SetContainerTitle(id, title string) {
	if ct, ok := v.containers.Get(id); ok {
		ct.SetTitle(title)
	}
}

func (v *View) UpdateTreeNodeTitle(id, title string) {
	if node, ok := v.treeViewNodes.Get(id); ok {
		node.Text = title
	}
}

func (v *View) SetTabDirty(id string, dirty bool) {
	if tab, ok := v.openTabs.Get(id); ok {
		tab.SetDataChanged(dirty)
		if ct, ok := v.containers.Get(id); ok {
			ct.SetDataChanged(dirty)
		}
	}
}

func (v *View) CloseTab(id string) {
	if _, ok := v.openTabs.Get(id); ok {
		v.tabHeader.RemoveTabByID(id)
		v.openTabs.Delete(id)
		v.containers.Delete(id)
	}
}

func (v *View) IsTabOpen(id string) bool {
	_, ok := v.openTabs.Get(id)
	return ok
}

func (v *View) SwitchToTab(id string) {
	if _, ok := v.openTabs.Get(id); ok {
		v.tabHeader.SetSelectedByID(id)
	}
}

func (v *View) OpenTab(id, name, tabType string) {
	tab := &widgets.Tab{
		Title:          name,
		Closable:       true,
		CloseClickable: &widget.Clickable{},
		Identifier:     id,
		Meta:           safemap.New[string](),
	}
	tab.Meta.Set(TypeMeta, tabType)

	tab.SetOnClose(func(tab *widgets.Tab) {
		if v.controller != nil {
			v.controller.OnTabClose(tab.Identifier)
		}
	})

	i := v.tabHeader.AddTab(tab)
	v.openTabs.Set(id, tab)
	v.tabHeader.SetSelected(i)
}

// TODO consider removing this methods and meta and make on click callback to provide the type

func (v *View) GetTabType(id string) string {
	if tab, ok := v.openTabs.Get(id); ok {
		if t, ok := tab.Meta.Get(TypeMeta); ok {
			return t
		}
	}

	return ""
}

func (v *View) GetTreeViewNodeType(id string) string {
	if tab, ok := v.treeViewNodes.Get(id); ok {
		if t, ok := tab.Meta.Get(TypeMeta); ok {
			return t
		}
	}

	return ""
}

func (v *View) OpenRequestContainer(req *domain.Request) {
	if _, ok := v.containers.Get(req.MetaData.ID); ok {
		return
	}

	if req.MetaData.Type == domain.RequestTypeHTTP {
		ct := v.createRestfulContainer(req)
		v.containers.Set(req.MetaData.ID, ct)
	}

	if req.MetaData.Type == domain.RequestTypeGRPC {
		ct := v.createGrpcContainer(req)
		v.containers.Set(req.MetaData.ID, ct)
	}

	if req.MetaData.Type == domain.RequestTypeGraphQL {
		ct := v.createGraphQLContainer(req)
		v.containers.Set(req.MetaData.ID, ct)
	}

	v.window.Invalidate()
}

func (v *View) createGrpcContainer(req *domain.Request) Container {
	ct := grpc.New(req, v.theme, v.explorer)

	ct.SetOnTitleChanged(func(text string) {
		if v.controller != nil {
			v.controller.OnTitleChanged(req.MetaData.ID, text, TypeRequest)
		}
	})

	ct.SetOnSave(func(id string) {
		if v.controller != nil {
			v.controller.OnSave(id)
		}
	})

	ct.SetOnDataChanged(func(id string, data any) {
		if v.controller != nil {
			v.controller.OnDataChanged(id, data, TypeRequest)
		}
	})

	ct.SetOnReload(func(id string) {
		if v.controller != nil {
			v.controller.OnServerInfoReload(id)
		}
	})

	ct.SetOnInvoke(func(id string) {
		if v.controller != nil {
			v.controller.OnGrpcInvoke(id)
		}
	})

	ct.SetOnSetOnTriggerRequestChanged(func(id, collectionID, requestID string) {
		if v.controller != nil {
			v.controller.OnSetOnTriggerRequestChanged(id, collectionID, requestID)
		}
	})

	ct.SetOnPostRequestSetChanged(func(id string, statusCode int, item, from, fromKey string) {
		if v.controller != nil {
			v.controller.OnPostRequestSetChanged(id, statusCode, item, from, fromKey)
		}
	})

	ct.SetOnLoadRequestExample(func(id string) {
		if v.controller != nil {
			v.controller.OnGrpcLoadRequestExample(id)
		}
	})

	ct.SetOnCopyResponse(func(gtx layout.Context, dataType, data string) {
		if v.controller != nil {
			v.controller.OnCopyResponse(gtx, dataType, data)
		}
	})

	ct.SetOnRequestTabChange(func(id, tab string) {
		if v.controller != nil {
			v.controller.OnRequestTabChanged(id, tab)
		}
	})

	ct.SetOnCreateCollectionFromMethods(func() {
		if v.controller != nil {
			v.controller.OnCreateCollectionFromMethods(req.MetaData.ID)
		}
	})

	return ct
}

func (v *View) createRestfulContainer(req *domain.Request) Container {
	ct := restful.New(req, v.theme, v.explorer)

	ct.SetOnTitleChanged(func(text string) {
		if v.controller != nil {
			v.controller.OnTitleChanged(req.MetaData.ID, text, TypeRequest)
		}
	})

	ct.SetOnSave(func(id string) {
		if v.controller != nil {
			v.controller.OnSave(id)
		}
	})

	ct.SetOnDataChanged(func(id string, data any) {
		if v.controller != nil {
			v.controller.OnDataChanged(id, req, TypeRequest)
		}
	})

	ct.SetOnSubmit(func(id string) {
		if v.controller != nil {
			v.controller.OnSubmit(id, TypeRequest)
		}
	})

	ct.SetOnCopyResponse(func(gtx layout.Context, dataType, data string) {
		if v.controller != nil {
			v.controller.OnCopyResponse(gtx, dataType, data)
		}
	})

	ct.SetOnPostRequestSetChanged(func(id string, statusCode int, item, from, fromKey string) {
		if v.controller != nil {
			v.controller.OnPostRequestSetChanged(id, statusCode, item, from, fromKey)
		}
	})

	ct.SetOnSetOnTriggerRequestChanged(func(id, collectionID, requestID string) {
		if v.controller != nil {
			v.controller.OnSetOnTriggerRequestChanged(id, collectionID, requestID)
		}
	})

	ct.SetOnBinaryFileSelect(func(id string) {
		if v.controller != nil {
			v.controller.OnBinaryFileSelect(id)
		}
	})

	ct.SetOnFormDataFileSelect(func(requestId, fieldId string) {
		if v.controller != nil {
			v.controller.OnFormDataFileSelect(requestId, fieldId)
		}
	})

	ct.SetOnRequestTabChange(func(id, tab string) {
		if v.controller != nil {
			v.controller.OnRequestTabChanged(id, tab)
		}
	})

	return ct
}

func (v *View) createGraphQLContainer(req *domain.Request) Container {
	ct := graphql.New(req, v.theme, v.explorer)

	ct.SetOnTitleChanged(func(text string) {
		if v.controller != nil {
			v.controller.OnTitleChanged(req.MetaData.ID, text, TypeRequest)
		}
	})

	ct.SetOnSave(func(id string) {
		if v.controller != nil {
			v.controller.OnSave(id)
		}
	})

	ct.SetOnDataChanged(func(id string, data any) {
		if v.controller != nil {
			v.controller.OnDataChanged(id, data, TypeRequest)
		}
	})

	ct.SetOnSubmit(func(id string) {
		if v.controller != nil {
			v.controller.OnSubmit(id, TypeRequest)
		}
	})

	ct.SetOnCopyResponse(func(gtx layout.Context, dataType, data string) {
		if v.controller != nil {
			v.controller.OnCopyResponse(gtx, dataType, data)
		}
	})

	ct.SetOnPostRequestSetChanged(func(id string, statusCode int, item, from, fromKey string) {
		if v.controller != nil {
			v.controller.OnPostRequestSetChanged(id, statusCode, item, from, fromKey)
		}
	})

	ct.SetOnSetOnTriggerRequestChanged(func(id, collectionID, requestID string) {
		if v.controller != nil {
			v.controller.OnSetOnTriggerRequestChanged(id, collectionID, requestID)
		}
	})

	ct.SetOnRequestTabChange(func(id, tab string) {
		if v.controller != nil {
			v.controller.OnRequestTabChanged(id, tab)
		}
	})

	return ct
}

func (v *View) SetRequestCollection(id string, collection *domain.Collection) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetCollection(collection)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetCollection(collection)
		}
		// TODO: Add support for gRPC containers if needed
	}
}

// UpdateCollectionForOpenRequests updates all open request containers that belong to the given collection
// It takes a function to check if a request ID belongs to the collection
func (v *View) UpdateCollectionForOpenRequests(collectionID string, collection *domain.Collection, belongsToCollection func(requestID string) bool) {
	updated := false
	// Iterate through all open tabs to find requests that belong to this collection
	for _, tabID := range v.openTabs.Keys() {
		tab, ok := v.openTabs.Get(tabID)
		if !ok {
			continue
		}

		// Check if this tab is a request
		tabType, ok := tab.Meta.Get(TypeMeta)
		if !ok || tabType != TypeRequest {
			continue
		}

		// Check if this request belongs to the collection
		if !belongsToCollection(tabID) {
			continue
		}

		// Update the container with the new collection data
		if ct, ok := v.containers.Get(tabID); ok {
			if ct, ok := ct.(RestContainer); ok {
				ct.SetCollection(collection)
				updated = true
			}
			// TODO: Add support for gRPC containers if needed
		}
	}

	// Invalidate window to trigger UI refresh if any containers were updated
	if updated {
		v.window.Invalidate()
	}
}

func (v *View) SetSendingRequestLoading(id string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.ShowSendingRequestLoading()
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetResponseLoading(true)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.ShowSendingRequestLoading()
		}
	}
}

func (v *View) SetSendingRequestLoaded(id string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.HideSendingRequestLoading()
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetResponseLoading(false)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.HideSendingRequestLoading()
		}
	}
}

func (v *View) SetQueryParams(id string, params []domain.KeyValue) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetQueryParams(params)
		}
	}
}

func (v *View) SetPathParams(id string, params []domain.KeyValue) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetPathParams(params)
		}
	}
}

func (v *View) SetURL(id, url string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetURL(url)
			return
		}

		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetURL(url)
		}
	}
}

func (v *View) OpenCollectionContainer(collection *domain.Collection) {
	if _, ok := v.containers.Get(collection.MetaData.ID); ok {
		return
	}

	ct := collections.New(collection, v.theme)
	ct.SetOnTitleChanged(func(text string) {
		if v.controller != nil {
			v.controller.OnTitleChanged(collection.MetaData.ID, text, TypeCollection)
		}
	})

	ct.SetOnDataChanged(func(id string, data any) {
		if v.controller != nil {
			v.controller.OnDataChanged(id, data, TypeCollection)
		}
	})

	ct.SetOnSave(func(id string) {
		if v.controller != nil {
			v.controller.OnSave(id)
		}
	})

	v.containers.Set(collection.MetaData.ID, ct)
}

func (v *View) SetHTTPResponse(id string, response domain.HTTPResponseDetail) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetHTTPResponse(response)
			v.window.Invalidate()
		}
	}
}

func (v *View) SetGRPCResponse(id string, response domain.GRPCResponseDetail) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetResponse(response)
			v.window.Invalidate()
		}
	}
}

func (v *View) SetGraphQLResponse(id string, response domain.GraphQLResponseDetail) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GraphQLContainer); ok {
			ct.SetGraphQLResponse(response)
			v.window.Invalidate()
		}
	}
}

func (v *View) GetHTTPResponse(id string) *domain.HTTPResponseDetail {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			return ct.GetHTTPResponse()
		}
	}

	return nil
}

func (v *View) GetGRPCResponse(id string) *domain.GRPCResponseDetail {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GrpcContainer); ok {
			return ct.GetResponse()
		}
	}

	return nil
}

func (v *View) GetGraphQLResponse(id string) *domain.GraphQLResponseDetail {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GraphQLContainer); ok {
			return ct.GetGraphQLResponse()
		}
	}

	return nil
}

func (v *View) ShowPrompt(id, title, content, modalType string, onSubmit func(selectedOption string, remember bool), options ...widgets.Option) {
	ct, ok := v.containers.Get(id)
	if !ok {
		return
	}

	ct.ShowPrompt(title, content, modalType, onSubmit, options...)
}

func (v *View) HideGRPCRequestError(id string) {
	ct, ok := v.containers.Get(id)
	if !ok {
		return
	}

	if ct, ok := ct.(GrpcContainer); ok {
		ct.HideRequestPrompt()
	}
}

func (v *View) ShowGRPCRequestError(id, title, content string) {
	ct, ok := v.containers.Get(id)
	if !ok {
		return
	}

	if ct, ok := ct.(GrpcContainer); ok {
		ct.ShowRequestPrompt(title, content, widgets.ModalTypeErr, func(selectedOption string, remember bool) {
			if selectedOption == "Ok" {
				ct.HideRequestPrompt()
				return
			}

		}, []widgets.Option{{Text: "Ok"}}...)
	}
}

func (v *View) HidePrompt(id string) {
	ct, ok := v.containers.Get(id)
	if !ok {
		return
	}

	ct.HidePrompt()
}

func (v *View) PopulateTreeView(requests []*domain.Request, collections []*domain.Collection) {
	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, cl := range collections {
		parentNode := &widgets.TreeNode{
			Text:        cl.MetaData.Name,
			Identifier:  cl.MetaData.ID,
			Children:    make([]*widgets.TreeNode, 0),
			MenuOptions: []string{MenuAddHTTPRequest, MenuAddGRPCRequest, MenuAddGraphQLRequest, MenuDuplicate, MenuView, MenuDelete},
			Meta:        safemap.New[string](),
		}
		parentNode.Meta.Set(TypeMeta, TypeCollection)

		for _, req := range cl.Spec.Requests {
			node := &widgets.TreeNode{
				Text:        req.MetaData.Name,
				Identifier:  req.MetaData.ID,
				MenuOptions: []string{MenuView, MenuDuplicate, MenuDelete},
				Meta:        safemap.New[string](),
			}

			setNodePrefix(req, node)

			node.Meta.Set(TypeMeta, TypeRequest)
			parentNode.AddChildNode(node)
			v.treeViewNodes.Set(req.MetaData.ID, node)
		}

		treeViewNodes = append(treeViewNodes, parentNode)
		v.treeViewNodes.Set(cl.MetaData.ID, parentNode)
	}

	for _, req := range requests {
		node := &widgets.TreeNode{
			Text:        req.MetaData.Name,
			Identifier:  req.MetaData.ID,
			MenuOptions: []string{MenuView, MenuDuplicate, MenuDelete},
			Meta:        safemap.New[string](),
		}

		setNodePrefix(req, node)

		node.Meta.Set(TypeMeta, TypeRequest)
		treeViewNodes = append(treeViewNodes, node)
		v.treeViewNodes.Set(req.MetaData.ID, node)
	}

	v.treeView.SetNodes(treeViewNodes)
}

func (v *View) AddTreeViewNode(req *domain.Request) {
	v.addTreeViewNode("", req)
}

func (v *View) AddChildTreeViewNode(parentID string, req *domain.Request) {
	v.addTreeViewNode(parentID, req)
}

func (v *View) SetTreeViewNodePrefix(id string, req *domain.Request) {
	if node, ok := v.treeViewNodes.Get(id); ok {
		setNodePrefix(req, node)
		v.window.Invalidate()
	}
}

func (v *View) addTreeViewNode(parentID string, req *domain.Request) {
	if req.MetaData.ID == "" {
		req.MetaData.ID = uuid.NewString()
	}

	node := &widgets.TreeNode{
		Text:        req.MetaData.Name,
		Identifier:  req.MetaData.ID,
		MenuOptions: []string{MenuDuplicate, MenuDelete},
		Meta:        safemap.New[string](),
	}

	setNodePrefix(req, node)

	node.Meta.Set(TypeMeta, TypeRequest)
	if parentID == "" {
		v.treeView.AddNode(node)
	} else {
		v.treeView.AddChildNode(parentID, node)
	}
	v.treeViewNodes.Set(req.MetaData.ID, node)
}

func setNodePrefix(req *domain.Request, node *widgets.TreeNode) {

	switch req.MetaData.Type {
	case domain.RequestTypeGRPC:
		node.Prefix = "gRPC"
		node.PrefixColor = chapartheme.GetRequestPrefixColor("gRPC")
	case domain.RequestTypeGraphQL:
		node.Prefix = "GraphQL"
		node.PrefixColor = chapartheme.GetRequestPrefixColor("GraphQL")
	case domain.RequestTypeHTTP:
		if req.Spec.HTTP != nil {
			node.Prefix = req.Spec.HTTP.Method
			node.PrefixColor = chapartheme.GetRequestPrefixColor(req.Spec.HTTP.Method)
		}
	default:
		node.Prefix = string(req.MetaData.Type)
		node.PrefixColor = chapartheme.GetRequestPrefixColor(string(req.MetaData.Type))
	}
}

func (v *View) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	return v.split.Layout(gtx, theme,
		func(gtx layout.Context) layout.Dimensions {
			return v.requestList(gtx, theme)
		},
		func(gtx layout.Context) layout.Dimensions {
			return v.containerHolder(gtx, theme)
		},
	)
}

func (v *View) requestList(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.treeViewSearchBox.Changed() {
		if v.treeViewNodes.Len() > 0 {
			v.treeView.Filter(v.treeViewSearchBox.GetText())
		}
	}

	if v.newRequestButton.Clicked(gtx) {
		v.showCreateNewModal()
	}

	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.importButton.Clicked(gtx) {
								v.showImportModal()
							}
							btn := widgets.Button(theme.Material(), &v.importButton, widgets.UploadIcon, widgets.IconPositionStart, "Import")
							btn.Color = theme.ButtonTextColor
							return btn.Layout(gtx, theme)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							newBtn := widgets.Button(theme.Material(), &v.newRequestButton, widgets.PlusIcon, widgets.IconPositionStart, "New")
							newBtn.Color = theme.ButtonTextColor
							return newBtn.Layout(gtx, theme)
						}),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeViewSearchBox.Layout(gtx, theme)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(10), Right: 0}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return v.treeView.Layout(gtx, theme)
				})
			}),
		)
	})
}

func (v *View) showCreateNewModal() {
	items := []*modals.CreateItem{newGRPCRequest, newHTTPRequest, newGraphQLRequest, newHCollection}

	m := modals.NewCreateModal(items)
	v.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.CloseBtn.Clicked(gtx) {
			v.Base.CloseModal()
		}

		for _, item := range items {
			if item.Clickable.Clicked(gtx) {
				v.Base.CloseModal()
				v.createNew(domain.RequestType(item.Key))
			}
		}

		return m.Layout(gtx, v.Theme)
	})
}

func (v *View) showImportModal() {
	m := modals.NewImportModal()
	v.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.CloseBtn.Clicked(gtx) {
			v.Base.CloseModal()
		}

		for _, item := range m.Items {
			if item.Clickable.Clicked(gtx) {
				v.Base.CloseModal()
				if v.controller != nil {
					v.controller.OnImport(item.Key)
				}
			}
		}

		return m.Layout(gtx, v.Theme)
	})
}

func (v *View) createNew(itemType domain.RequestType) {
	if v.controller == nil {
		return
	}

	switch itemType {
	case domain.RequestTypeGRPC:
		v.controller.OnNewRequest(domain.RequestTypeGRPC)
	case domain.RequestTypeHTTP:
		v.controller.OnNewRequest(domain.RequestTypeHTTP)
	case domain.RequestTypeGraphQL:
		v.controller.OnNewRequest(domain.RequestTypeGraphQL)
	case domain.KindCollection:
		v.controller.OnNewCollection()
	}
}

func (v *View) containerHolder(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.controller != nil {
		keys.OnSaveCommand(gtx, v, func() {
			v.controller.OnSave(v.tabHeader.SelectedTab().GetIdentifier())
		})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return v.tabHeader.Layout(gtx, theme)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if v.openTabs.Len() == 0 {
				return v.tipsView.Layout(gtx, theme)
			}

			selectedTab := v.tabHeader.SelectedTab()
			if selectedTab != nil {
				if ct, ok := v.containers.Get(selectedTab.Identifier); ok {
					return ct.Layout(gtx, theme)
				}
			}

			return layout.Dimensions{}
		}),
	)
}
