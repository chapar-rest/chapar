package requests

import (
	"image"

	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/pages/requests/grpc"

	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/x/component"
	giox "gioui.org/x/component"
	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/chapartheme"
	"github.com/chapar-rest/chapar/ui/keys"
	"github.com/chapar-rest/chapar/ui/pages/requests/collections"
	"github.com/chapar-rest/chapar/ui/pages/requests/restful"
	"github.com/chapar-rest/chapar/ui/pages/tips"
	"github.com/chapar-rest/chapar/ui/widgets"
)

const (
	MenuDuplicate  = "Duplicate"
	MenuDelete     = "Delete"
	MenuAddRequest = "Add Request"
	MenuView       = "View"
)

type View struct {
	theme  *chapartheme.Theme
	window *app.Window

	modal *component.ModalLayer
	// add menu
	newRequestButton     widget.Clickable
	importButton         widget.Clickable
	newMenuContextArea   giox.ContextArea
	newMenu              giox.MenuState
	menuInit             bool
	newHttpRequestButton widget.Clickable
	newGrpcRequestButton widget.Clickable
	newCollectionButton  widget.Clickable

	treeViewSearchBox *widgets.TextField
	treeView          *widgets.TreeView

	split     widgets.SplitView
	tabHeader *widgets.Tabs

	// callbacks
	onTitleChanged              func(id, title, containerType string)
	onNewRequest                func(requestType string)
	onImport                    func()
	onNewCollection             func()
	onTabClose                  func(id string)
	onTreeViewNodeDoubleClicked func(id string)
	onTreeViewNodeClicked       func(id string)
	onTreeViewMenuClicked       func(id string, action string)
	onTabSelected               func(id string)
	onSave                      func(id string)
	onSubmit                    func(id, containerType string)
	onDataChanged               func(id string, data any, containerType string)
	onCopyResponse              func(gtx layout.Context, dataType, data string)
	onOnPostRequestSetChanged   func(id string, statusCode int, item, from, fromKey string)
	onBinaryFileSelect          func(id string)
	onProtoFileSelect           func(id string)
	onFromDataFileSelect        func(requestID, fieldID string)
	onServerInfoReload          func(id string)
	onGrpcInvoke                func(id string)
	onGrpcLoadRequestExample    func(id string)

	// state
	containers    *safemap.Map[Container]
	openTabs      *safemap.Map[*widgets.Tab]
	treeViewNodes *safemap.Map[*widgets.TreeNode]

	tipsView *tips.Tips
}

func NewView(w *app.Window, theme *chapartheme.Theme) *View {
	search := widgets.NewTextField("", "Search...")
	search.SetIcon(widgets.SearchIcon, widgets.IconPositionEnd)

	v := &View{
		window:            w,
		theme:             theme,
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
		newMenuContextArea: component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		},

		modal:    component.NewModal(),
		tipsView: tips.New(),
	}

	v.modal.Widget = func(gtx layout.Context, th *material.Theme, anim *component.VisibilityAnimation) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return component.Surface(th).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(th, unit.Sp(16), "Hello, World!").Layout(gtx)
			})
		})
	}

	v.tabHeader.SetMaxTitleWidth(20)

	v.treeViewSearchBox.SetOnTextChange(func(text string) {
		if v.treeViewNodes.Len() == 0 {
			return
		}
		v.treeView.Filter(text)
	})

	return v
}

func (v *View) SetOnPostRequestSetChanged(f func(id string, statusCode int, item, from, fromKey string)) {
	v.onOnPostRequestSetChanged = f
}

func (v *View) SetPostRequestSetValues(id string, set domain.PostRequestSet) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.SetPostRequestSetValues(set)
		}
	}
}

func (v *View) SetPostRequestSetPreview(id, preview string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
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
		MenuOptions: []string{MenuAddRequest, MenuView, MenuDelete},
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

func (v *View) SetOnCopyResponse(onCopyResponse func(gtx layout.Context, dataType, data string)) {
	v.onCopyResponse = onCopyResponse
}

func (v *View) SetOnBinaryFileSelect(f func(id string)) {
	v.onBinaryFileSelect = f
}

func (v *View) SetOnProtoFileSelect(f func(id string)) {
	v.onProtoFileSelect = f
}

func (v *View) SetOnServerInfoReload(f func(id string)) {
	v.onServerInfoReload = f
}

func (v *View) SetOnGrpcInvoke(f func(id string)) {
	v.onGrpcInvoke = f
}

func (v *View) SetOnGrpcLoadRequestExample(f func(id string)) {
	v.onGrpcLoadRequestExample = f
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

func (v *View) SetProtoFilePath(id, filePath string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetProtoBodyFilePath(filePath)
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
			if loading {
				ct.ShowMethodsLoading()
			} else {
				ct.HideMethodsLoading()
			}
		}
	}
}

func (v *View) SetOnFormDataFileSelect(f func(requestId, fieldId string)) {
	v.onFromDataFileSelect = f
}

func (v *View) AddFileToFormData(requestId, fieldId, filePath string) {
	if ct, ok := v.containers.Get(requestId); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.AddFileToFormData(fieldId, filePath)
		}
	}
}

func (v *View) SetOnNewRequest(onNewRequest func(requestType string)) {
	v.onNewRequest = onNewRequest
}

func (v *View) SetOnDataChanged(onDataChanged func(id string, data any, containerType string)) {
	v.onDataChanged = onDataChanged
}

func (v *View) SetOnNewCollection(onNewCollection func()) {
	v.onNewCollection = onNewCollection
}

func (v *View) SetOnSubmit(f func(id, containerType string)) {
	v.onSubmit = f
}

func (v *View) SetOnImport(f func()) {
	v.onImport = f
}

func (v *View) SetOnTitleChanged(onTitleChanged func(id, title, containerType string)) {
	v.onTitleChanged = onTitleChanged
}

func (v *View) SetOnTreeViewNodeDoubleClicked(onTreeViewNodeDoubleClicked func(id string)) {
	v.onTreeViewNodeDoubleClicked = onTreeViewNodeDoubleClicked
	v.treeView.OnNodeDoubleClick(func(node *widgets.TreeNode) {
		v.onTreeViewNodeDoubleClicked(node.Identifier)
	})
}

func (v *View) SetOnTreeViewNodeClicked(onTreeViewNodeClicked func(id string)) {
	v.onTreeViewNodeClicked = onTreeViewNodeClicked
	v.treeView.OnNodeClick(func(node *widgets.TreeNode) {
		v.onTreeViewNodeClicked(node.Identifier)
	})
}

func (v *View) SetOnTreeViewMenuClicked(onTreeViewMenuClicked func(id, action string)) {
	v.onTreeViewMenuClicked = onTreeViewMenuClicked
	v.treeView.SetOnMenuItemClick(func(node *widgets.TreeNode, item string) {
		v.onTreeViewMenuClicked(node.Identifier, item)
	})
}

func (v *View) SetOnTabSelected(onTabSelected func(id string)) {
	v.onTabSelected = onTabSelected
}

func (v *View) SetOnSave(onSave func(id string)) {
	v.onSave = onSave
}

func (v *View) ExpandTreeViewNode(id string) {
	node, ok := v.treeViewNodes.Get(id)
	if !ok {
		return
	}

	v.treeView.ExpandNode(node.Identifier)
}

func (v *View) SetOnTabClose(onTabClose func(id string)) {
	v.onTabClose = onTabClose
}

func (v *View) UpdateTabTitle(id, title string) {
	if tab, ok := v.openTabs.Get(id); ok {
		tab.Title = title
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

	if v.onTabClose != nil {
		tab.SetOnClose(func(tab *widgets.Tab) {
			v.onTabClose(tab.Identifier)
		})
	}

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
		return
	}

	if req.MetaData.Type == domain.RequestTypeGRPC {
		ct := v.createGrpcContainer(req)
		v.containers.Set(req.MetaData.ID, ct)
		return
	}
}

func (v *View) createGrpcContainer(req *domain.Request) Container {
	ct := grpc.New(req, v.theme)

	ct.SetOnTitleChanged(func(text string) {
		if v.onTitleChanged != nil {
			v.onTitleChanged(req.MetaData.ID, text, TypeRequest)
		}
	})

	ct.SetOnSave(func(id string) {
		if v.onSave != nil {
			v.onSave(id)
		}
	})

	ct.SetOnDataChanged(func(id string, data any) {
		if v.onDataChanged != nil {
			v.onDataChanged(id, data, TypeRequest)
		}
	})

	ct.SetOnProtoFileSelect(func(id string) {
		if v.onProtoFileSelect != nil {
			v.onProtoFileSelect(id)
		}
	})

	ct.SetOnReload(func(id string) {
		if v.onServerInfoReload != nil {
			v.onServerInfoReload(id)
		}
	})

	ct.SetOnInvoke(func(id string) {
		if v.onGrpcInvoke != nil {
			v.onGrpcInvoke(id)
		}
	})

	ct.SetOnLoadRequestExample(func(id string) {
		if v.onGrpcLoadRequestExample != nil {
			v.onGrpcLoadRequestExample(id)
		}
	})

	ct.SetOnCopyResponse(func(gtx layout.Context, dataType, data string) {
		if v.onCopyResponse != nil {
			v.onCopyResponse(gtx, dataType, data)
		}
	})

	return ct
}

func (v *View) createRestfulContainer(req *domain.Request) Container {
	ct := restful.New(req, v.theme)

	ct.SetOnTitleChanged(func(text string) {
		if v.onTitleChanged != nil {
			v.onTitleChanged(req.MetaData.ID, text, TypeRequest)
		}
	})

	ct.SetOnSave(func(id string) {
		if v.onSave != nil {
			v.onSave(id)
		}
	})

	ct.SetOnDataChanged(func(id string, data any) {
		if v.onDataChanged != nil {
			v.onDataChanged(id, req, TypeRequest)
		}
	})

	ct.SetOnSubmit(func(id string) {
		if v.onSubmit != nil {
			v.onSubmit(id, TypeRequest)
		}
	})

	ct.SetOnCopyResponse(func(gtx layout.Context, dataType, data string) {
		if v.onCopyResponse != nil {
			v.onCopyResponse(gtx, dataType, data)
		}
	})

	ct.SetOnPostRequestSetChanged(func(id string, statusCode int, item, from, fromKey string) {
		if v.onOnPostRequestSetChanged != nil {
			v.onOnPostRequestSetChanged(id, statusCode, item, from, fromKey)
		}
	})

	ct.SetOnBinaryFileSelect(func(id string) {
		if v.onBinaryFileSelect != nil {
			v.onBinaryFileSelect(id)
		}
	})

	ct.SetOnFormDataFileSelect(func(requestId, fieldId string) {
		if v.onFromDataFileSelect != nil {
			v.onFromDataFileSelect(requestId, fieldId)
		}
	})

	return ct
}

func (v *View) SetSendingRequestLoading(id string) {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			ct.ShowSendingRequestLoading()
			return
		}

		if ct, ok := ct.(GrpcContainer); ok {
			ct.SetResponseLoading(true)
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
		}
	}
}

func (v *View) OpenCollectionContainer(collection *domain.Collection) {
	if _, ok := v.containers.Get(collection.MetaData.ID); ok {
		return
	}

	ct := collections.New(collection)
	ct.Title.SetOnChanged(func(text string) {
		if v.onTitleChanged != nil {
			v.onTitleChanged(collection.MetaData.ID, text, TypeCollection)
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

func (v *View) GetHTTPResponse(id string) *domain.HTTPResponseDetail {
	if ct, ok := v.containers.Get(id); ok {
		if ct, ok := ct.(RestContainer); ok {
			return ct.GetHTTPResponse()
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
			MenuOptions: []string{MenuAddRequest, MenuView, MenuDelete},
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
	if req.MetaData.Type == domain.RequestTypeGRPC {
		node.Prefix = "gRPC"
		node.PrefixColor = chapartheme.GetRequestPrefixColor("gRPC")
	} else {
		node.Prefix = req.Spec.HTTP.Method
		node.PrefixColor = chapartheme.GetRequestPrefixColor(req.Spec.HTTP.Method)
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
	if !v.menuInit {
		v.menuInit = true
		v.newMenu = component.MenuState{
			Options: []func(gtx layout.Context) layout.Dimensions{
				component.MenuItem(theme.Material(), &v.newHttpRequestButton, "Restful Request").Layout,
				component.MenuItem(theme.Material(), &v.newGrpcRequestButton, "GRPC Request").Layout,
				component.Divider(theme.Material()).Layout,
				component.MenuItem(theme.Material(), &v.newCollectionButton, "Collection").Layout,
			},
		}
	}

	if v.newHttpRequestButton.Clicked(gtx) {
		if v.onNewRequest != nil {
			v.onNewRequest(domain.RequestTypeHTTP)
		}
	}

	if v.newGrpcRequestButton.Clicked(gtx) {
		if v.onNewRequest != nil {
			v.onNewRequest(domain.RequestTypeGRPC)
		}
	}

	if v.newCollectionButton.Clicked(gtx) {
		if v.onNewCollection != nil {
			v.onNewCollection()
		}
	}

	return layout.Inset{Top: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceStart}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if v.importButton.Clicked(gtx) {
								if v.onImport != nil {
									v.onImport()
								}
							}
							btn := widgets.Button(theme.Material(), &v.importButton, widgets.UploadIcon, widgets.IconPositionStart, "Import")
							btn.Color = theme.ButtonTextColor
							return btn.Layout(gtx, theme)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							newBtn := widgets.Button(theme.Material(), &v.newRequestButton, widgets.PlusIcon, widgets.IconPositionStart, "New")
							newBtn.Color = theme.ButtonTextColor
							newBtnDims := newBtn.Layout(gtx, theme)
							return layout.Stack{}.Layout(gtx,
								layout.Stacked(func(gtx layout.Context) layout.Dimensions {
									return newBtnDims
								}),
								layout.Expanded(func(gtx layout.Context) layout.Dimensions {
									return v.newMenuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										offset := layout.Inset{
											Top:  unit.Dp(float32(newBtnDims.Size.Y)/gtx.Metric.PxPerDp + 1),
											Left: unit.Dp(1),
										}
										return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											gtx.Constraints.Min = image.Point{}
											m := component.Menu(theme.Material(), &v.newMenu)
											m.SurfaceStyle.Fill = theme.MenuBgColor
											return m.Layout(gtx)
										})
									})
								}),
							)
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

func (v *View) containerHolder(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if v.onSave != nil {
		keys.OnSaveCommand(gtx, v, func() {
			v.onSave(v.tabHeader.SelectedTab().GetIdentifier())
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
				if v.onTabSelected != nil {
					v.onTabSelected(selectedTab.Identifier)
				}

				if ct, ok := v.containers.Get(selectedTab.Identifier); ok {
					return ct.Layout(gtx, theme)
				}
			}

			return layout.Dimensions{}
		}),
	)
}
