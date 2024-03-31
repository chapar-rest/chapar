package requests

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mirzakhany/chapar/internal/notify"

	"gioui.org/io/clipboard"

	"gioui.org/layout"

	"github.com/mirzakhany/chapar/internal/repository"

	"github.com/mirzakhany/chapar/internal/state"

	"github.com/mirzakhany/chapar/internal/rest"

	"github.com/mirzakhany/chapar/internal/logger"

	"github.com/mirzakhany/chapar/ui/widgets"

	"github.com/mirzakhany/chapar/internal/domain"
)

type Controller struct {
	model *state.Requests
	view  *View

	envState *state.Environments

	repo repository.Repository

	activeTabID string
}

func NewController(view *View, repo repository.Repository, model *state.Requests, envState *state.Environments) *Controller {
	c := &Controller{
		view:     view,
		model:    model,
		repo:     repo,
		envState: envState,
	}

	view.SetOnNewRequest(c.onNewRequest)
	view.SetOnNewCollection(c.onNewCollection)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeClicked(c.onTreeViewNodeClicked)
	view.SetOnTreeViewMenuClicked(c.onTreeViewMenuClicked)
	view.SetOnTabClose(c.onTabClose)
	view.SetOnDataChanged(c.onDataChanged)
	view.SetOnSave(c.onSave)
	view.SetOnSubmit(c.onSubmit)
	view.SetOnCopyResponse(c.onCopyResponse)
	return c
}

func (c *Controller) LoadData() error {
	collections, err := c.model.LoadCollectionsFromDisk()
	if err != nil {
		return err
	}

	requests, err := c.model.LoadRequestsFromDisk()
	if err != nil {
		return err
	}

	c.view.PopulateTreeView(requests, collections)
	return nil
}

func (c *Controller) onTitleChanged(id string, title, containerType string) {
	switch containerType {
	case TypeRequest:
		c.onRequestTitleChange(id, title)
	case TypeCollection:
		c.onCollectionTitleChange(id, title)
	}
}

func (c *Controller) onSave(id string) {
	tabType := c.view.GetTabType(id)
	if tabType == TypeRequest {
		c.saveRequestToDisc(id)
	}
}

func (c *Controller) onSubmit(id, containerType string) {
	if containerType == TypeRequest {
		c.onSubmitRequest(id)
	}
}

func (c *Controller) onCopyResponse(gtx layout.Context, response string) {
	gtx.Execute(clipboard.WriteCmd{
		Data: io.NopCloser(strings.NewReader(response)),
	})

	notify.Send("Response copied to clipboard", 2*time.Second)
}

func (c *Controller) onSubmitRequest(id string) {
	c.view.SetSendingRequestLoading(id)
	defer c.view.SetSendingRequestLoaded(id)

	res, err := rest.SendRequest(c.model.GetRequest(id).Spec.HTTP, c.envState.GetActiveEnvironment())
	if err != nil {
		fmt.Println("failed to send request", err)
		return
	}

	resp := string(res.Body)
	if res.IsJSON {
		resp = res.JSON
	}

	c.view.SetHTTPResponse(id, resp, mapToKeyValue(res.Headers), cookieToKeyValue(res.Cookies), res.StatusCode, res.TimePassed, len(res.Body))

}

func cookieToKeyValue(cookies []*http.Cookie) []domain.KeyValue {
	var kvs []domain.KeyValue
	for _, c := range cookies {
		kvs = append(kvs, domain.KeyValue{Key: c.Name, Value: c.Value})
	}
	return kvs
}

func mapToKeyValue(m map[string]string) []domain.KeyValue {
	var kvs []domain.KeyValue
	for k, v := range m {
		kvs = append(kvs, domain.KeyValue{Key: k, Value: v})
	}
	return kvs
}

func (c *Controller) onTabClose(id string) {
	// get Tab to check if it's a request or collection
	tabType := c.view.GetTabType(id)
	if tabType == TypeRequest {
		c.onRequestTabClose(id)
	}

	if tabType == TypeCollection {
		c.onCollectionTabClose(id)
	}
}

func (c *Controller) onCollectionTabClose(id string) {
	c.view.CloseTab(id)
}

func (c *Controller) onDataChanged(id string, data any, containerType string) {
	switch containerType {
	case TypeRequest:
		c.onRequestDataChanged(id, data)
	case TypeCollection:
		c.onCollectionDataChanged(id)
	}
}

func (c *Controller) onRequestDataChanged(id string, data any) {
	req := c.model.GetRequest(id)
	if req == nil {
		fmt.Println("failed to get request", id)
		return
	}

	inComingRequest, ok := data.(*domain.Request)
	if !ok {
		panic("failed to convert data to Request")
		return
	}

	// is data changed?
	if domain.CompareRequests(req, inComingRequest) {
		return
	}

	queryParamsChanged := !domain.CompareKeyValues(req.Spec.HTTP.Request.QueryParams, inComingRequest.Spec.HTTP.Request.QueryParams)
	urlChanged := inComingRequest.Spec.HTTP.URL != req.Spec.HTTP.URL

	// if query params and url are changed, update the url base on the query params
	if (queryParamsChanged && urlChanged) || queryParamsChanged {
		newURL := c.getNewURLWithParams(inComingRequest.Spec.HTTP.Request.QueryParams, inComingRequest.Spec.HTTP.URL)
		c.view.SetURL(id, newURL)
		inComingRequest.Spec.HTTP.URL = newURL
	} else if urlChanged {
		// update query params based on the new url
		newParams := c.getUrlParams(inComingRequest.Spec.HTTP.URL)
		c.view.SetQueryParams(id, newParams)
		inComingRequest.Spec.HTTP.Request.QueryParams = newParams

		// update the path params based on the new url
		newPathParams := domain.ParsePathParams(inComingRequest.Spec.HTTP.URL)
		c.view.SetPathParams(id, newPathParams)
		inComingRequest.Spec.HTTP.Request.PathParams = newPathParams
	}

	// break the reference
	clone := inComingRequest.Clone()
	req.Spec = clone.Spec

	if err := c.model.UpdateRequest(req, true); err != nil {
		fmt.Println("failed to update request", err)
		return
	}

	// set tab dirty if the in memory data is different from the file
	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		fmt.Println("failed to get request from file", err)
		return
	}
	c.view.SetTabDirty(id, !domain.CompareRequests(req, reqFromFile))
}

func (c *Controller) getNewURLWithParams(params []domain.KeyValue, url string) string {
	if len(params) == 0 {
		return url
	}

	// sync url params with url params editor
	if url == "" {
		return ""
	}

	urlParams := strings.Split(url, "?")
	if len(urlParams) < 2 {
		return url + "?" + domain.EncodeQueryParams(params)
	}

	return urlParams[0] + "?" + domain.EncodeQueryParams(params)
}

func (c *Controller) getUrlParams(newURL string) []domain.KeyValue {
	// sync url params with url params editor
	if newURL == "" {
		return nil
	}

	urlParams := strings.Split(newURL, "?")
	if len(urlParams) < 2 {
		return nil
	}

	return domain.ParseQueryParams(urlParams[1])
}

func (c *Controller) onCollectionDataChanged(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		fmt.Println("failed to get collection", id)
		return
	}
}

func (c *Controller) onRequestTabClose(id string) {
	// is tab data changed?
	// if yes show prompt
	// if no close tab
	req := c.model.GetRequest(id)
	if req == nil {
		fmt.Println("failed to get request", id)
		return
	}

	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		fmt.Println("failed to get environment from file", err)
		return
	}

	// if data is not changed close the tab
	if domain.CompareRequests(req, reqFromFile) {
		c.view.CloseTab(id)
		return
	}

	// TODO check user preference to remember the choice

	c.view.ShowPrompt(id, "Save", "Do you want to save the changes? (Tips: you can always save the changes using CMD/CTRL+s)", widgets.ModalTypeWarn,
		func(selectedOption string, remember bool) {
			if selectedOption == "Cancel" {
				c.view.HidePrompt(id)
				return
			}

			if selectedOption == "Yes" {
				c.saveRequestToDisc(id)
			}

			c.view.CloseTab(id)
			c.model.ReloadRequestFromDisc(id)
		},
		[]widgets.Option{{Text: "Yes"}, {Text: "No"}, {Text: "Cancel"}}...,
	)
}

func (c *Controller) onRequestTitleChange(id, title string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	if req.MetaData.Name == title {
		return
	}

	req.MetaData.Name = title

	if err := c.model.UpdateRequest(req, false); err != nil {
		fmt.Println("failed to update request", err)
		return
	}

	c.view.UpdateTreeNodeTitle(req.MetaData.ID, req.MetaData.Name)
	c.view.UpdateTabTitle(req.MetaData.ID, req.MetaData.Name)
}

func (c *Controller) onCollectionTitleChange(id, title string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	if col.MetaData.Name == title {
		return
	}

	col.MetaData.Name = title

	if err := c.model.UpdateCollection(col, false); err != nil {
		fmt.Println("failed to update collection", err)
		return
	}

	c.view.UpdateTreeNodeTitle(col.MetaData.ID, col.MetaData.Name)
	c.view.UpdateTabTitle(col.MetaData.ID, col.MetaData.Name)
}

func (c *Controller) onNewRequest() {
	req := domain.NewRequest("New Request")

	newFilePath, err := c.repo.GetNewRequestFilePath(req.MetaData.Name)
	if err != nil {
		fmt.Println("failed to get new file path", err)
		return
	}

	req.FilePath = newFilePath.Path
	req.MetaData.Name = newFilePath.NewName

	c.model.AddRequest(req)
	c.view.AddRequestTreeViewNode(req)
	c.saveRequestToDisc(req.MetaData.ID)
	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	c.view.OpenRequestContainer(req)
	c.view.SwitchToTab(req.MetaData.ID)
}

func (c *Controller) onNewCollection() {
	col := domain.NewCollection("New Collection")

	dirPath, err := c.repo.GetNewCollectionDir(col.MetaData.Name)
	if err != nil {
		fmt.Println("failed to get new collection dir", err)
		return
	}

	col.FilePath = dirPath.Path
	col.MetaData.Name = dirPath.NewName

	c.model.AddCollection(col)
	c.view.AddCollectionTreeViewNode(col)
	c.saveCollectionToDisc(col.MetaData.ID)
	c.view.OpenTab(col.MetaData.ID, col.MetaData.Name, TypeCollection)
	c.view.OpenCollectionContainer(col)
	c.view.SwitchToTab(col.MetaData.ID)
}

func (c *Controller) saveRequestToDisc(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}
	if err := c.model.UpdateRequest(req, false); err != nil {
		fmt.Println("failed to update request", err)
		return
	}
	c.view.SetTabDirty(id, false)
}

func (c *Controller) saveCollectionToDisc(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}
	if err := c.model.UpdateCollection(col, false); err != nil {
		fmt.Println("failed to update collection", err)
		return
	}
	c.view.SetTabDirty(id, false)
}

func (c *Controller) onTreeViewNodeClicked(id string) {
	c.viewRequest(id)
}

func (c *Controller) onTreeViewMenuClicked(id, action string) {
	nodeType := c.view.GetTreeViewNodeType(id)
	if nodeType == "" {
		return
	}

	switch action {
	case MenuDuplicate:
		c.duplicateRequest(id)
	case MenuDelete:
		switch nodeType {
		case TypeRequest:
			c.deleteRequest(id)
		case TypeCollection:
			c.deleteCollection(id)
		}
	case MenuAddRequest:
		c.addRequestToCollection(id)
	case MenuView:
		if nodeType == TypeCollection {
			c.viewCollection(id)
		} else {
			c.viewRequest(id)
		}
	}
}

func (c *Controller) addRequestToCollection(id string) {
	req := domain.NewRequest("New Request")
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	newFilePath, err := c.repo.GetCollectionRequestNewFilePath(col, req.MetaData.Name)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get new file path, err %v", err))
		return
	}

	req.FilePath = newFilePath.Path
	req.MetaData.Name = newFilePath.NewName
	req.CollectionID = col.MetaData.ID
	req.CollectionName = col.MetaData.Name

	c.model.AddRequest(req)
	c.model.AddRequestToCollection(col, req)
	c.saveRequestToDisc(req.MetaData.ID)

	c.view.AddChildTreeViewNode(col.MetaData.ID, req)
	c.view.ExpandTreeViewNode(col.MetaData.ID)
	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	c.view.OpenRequestContainer(req)
	c.view.SwitchToTab(req.MetaData.ID)
}

func (c *Controller) viewRequest(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	if c.view.IsTabOpen(id) {
		c.view.SwitchToTab(req.MetaData.ID)
		c.view.OpenRequestContainer(req)
		return
	}

	// make a clone to keep the original request unchanged
	clone, _ := domain.Clone[domain.Request](req)
	clone.MetaData.ID = id

	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	c.view.OpenRequestContainer(clone)
}

func (c *Controller) viewCollection(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	if c.view.IsTabOpen(id) {
		c.view.SwitchToTab(col.MetaData.ID)
		c.view.OpenCollectionContainer(col)
		return
	}

	c.view.OpenTab(col.MetaData.ID, col.MetaData.Name, TypeCollection)
	c.view.OpenCollectionContainer(col)
}

func (c *Controller) duplicateRequest(id string) {
	// read request from file to make sure we have the latest persisted data
	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		fmt.Println("failed to get request from file", err)
		return
	}

	newReq := reqFromFile.Clone()
	newReq.MetaData.Name = newReq.MetaData.Name + " (copy)"
	newReq.FilePath = repository.AddSuffixBeforeExt(newReq.FilePath, "-copy")
	c.model.AddRequest(newReq)
	if reqFromFile.CollectionID == "" {
		c.view.AddRequestTreeViewNode(newReq)
	} else {
		c.view.AddChildTreeViewNode(reqFromFile.CollectionID, newReq)
	}
	c.saveRequestToDisc(newReq.MetaData.ID)
}

func (c *Controller) deleteRequest(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}
	if err := c.model.RemoveRequest(req, false); err != nil {
		fmt.Println("failed to remove request", err)
		return
	}

	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
}

func (c *Controller) deleteCollection(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}
	if err := c.model.RemoveCollection(col, false); err != nil {
		fmt.Println("failed to remove collection", err)
		return
	}
	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
}
