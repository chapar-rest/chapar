package requests

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/layout"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/rest"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Controller struct {
	model *state.Requests
	view  *View

	envState *state.Environments

	repo repository.Repository

	explorer *explorer.Explorer

	restService *rest.Service
	grpcService *grpc.Service
}

func NewController(view *View, repo repository.Repository, model *state.Requests, envState *state.Environments, explorer *explorer.Explorer, restService *rest.Service, grpcService *grpc.Service) *Controller {
	c := &Controller{
		view:     view,
		model:    model,
		repo:     repo,
		envState: envState,

		explorer: explorer,

		restService: restService,
		grpcService: grpcService,
	}

	view.SetOnNewRequest(c.onNewRequest)
	view.SetOnImport(c.onImport)
	view.SetOnNewCollection(c.onNewCollection)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeClicked(c.onTreeViewNodeClicked)
	view.SetOnTreeViewMenuClicked(c.onTreeViewMenuClicked)
	view.SetOnTabClose(c.onTabClose)
	view.SetOnDataChanged(c.onDataChanged)
	view.SetOnSave(c.onSave)
	view.SetOnSubmit(c.onSubmit)
	view.SetOnCopyResponse(c.onCopyResponse)
	view.SetOnBinaryFileSelect(c.onSelectBinaryFile)
	view.SetOnProtoFileSelect(c.onProtoFileSelect)
	view.SetOnPostRequestSetChanged(c.onPostRequestSetChanged)
	view.SetOnFormDataFileSelect(c.onFormDataFileSelect)
	view.SetOnServerInfoReload(c.onServerInfoReload)
	view.SetOnGrpcInvoke(c.onGrpcInvoke)
	view.SetOnGrpcLoadRequestExample(c.onLoadRequestExample)
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

func (c *Controller) onSelectBinaryFile(id string) {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Error != nil {
			c.view.showError(fmt.Errorf("failed to get file, %w", result.Error))
			return
		}
		if result.FilePath == "" {
			return
		}
		c.view.SetBinaryBodyFilePath(id, result.FilePath)
	}, "")
}

func (c *Controller) onFormDataFileSelect(requestId, fieldId string) {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Error != nil {
			c.view.showError(fmt.Errorf("failed to get file, %w", result.Error))
			return
		}
		if result.FilePath == "" {
			return
		}
		c.view.AddFileToFormData(requestId, fieldId, result.FilePath)

	}, "")
}

func (c *Controller) getActiveEnvID() string {
	activeEnvironment := c.envState.GetActiveEnvironment()
	if activeEnvironment == nil {
		return ""
	}
	return activeEnvironment.MetaData.ID
}

func (c *Controller) onServerInfoReload(id string) {
	c.view.SetGRPCMethodsLoading(id, true)
	defer c.view.SetGRPCMethodsLoading(id, false)

	res, err := c.grpcService.GetServices(id, c.getActiveEnvID())
	if err != nil {
		c.view.ShowGRPCRequestError(id, "Failed to get server reflection", err.Error())
		return
	}

	c.view.HideGRPCRequestError(id)
	c.view.SetGRPCServices(id, res)
}

func (c *Controller) onGrpcInvoke(id string) {
	c.view.SetSendingRequestLoading(id)
	defer c.view.SetSendingRequestLoaded(id)

	resp, err := c.grpcService.Invoke(id, c.getActiveEnvID())
	if err != nil {
		c.view.SetGRPCResponse(id, domain.GRPCResponseDetail{
			Error: err,
		})
		return
	}

	c.view.SetGRPCResponse(id, domain.GRPCResponseDetail{
		Response:   resp.Body,
		Metadata:   resp.Metadata,
		Trailers:   resp.Trailers,
		StatusCode: resp.StatueCode,
		Duration:   resp.TimePassed,
		Status:     resp.Status,
		Size:       resp.Size,
	})
}

func (c *Controller) onLoadRequestExample(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	example, err := c.grpcService.GetRequestStruct(id, c.getActiveEnvID())
	if err != nil {
		c.view.ShowGRPCRequestError(id, "Failed to get request struct", err.Error())
		return
	}

	c.view.HideGRPCRequestError(id)
	c.view.SetSetGrpcRequestBody(id, example)
}

func (c *Controller) onProtoFileSelect(id string) {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Error != nil {
			c.view.ShowGRPCRequestError(id, "Error", result.Error.Error())
			return
		}
		c.view.HideGRPCRequestError(id)

		if result.FilePath == "" {
			return
		}
		c.view.SetProtoFilePath(id, result.FilePath)

	}, ".proto")
}

func (c *Controller) onPostRequestSetChanged(id string, statusCode int, item, from, fromKey string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	// break the reference
	clone := req.Clone()
	clone.MetaData.ID = id

	clone.Spec.HTTP.Request.PostRequest.PostRequestSet = domain.PostRequestSet{
		Target:     item,
		StatusCode: statusCode,
		From:       from,
		FromKey:    fromKey,
	}
	c.onRequestDataChanged(id, clone)

	responseData := c.view.GetHTTPResponse(id)
	if responseData == nil {
		return
	}

	if responseData.Response == "" {
		return
	}

	switch clone.Spec.HTTP.Request.PostRequest.PostRequestSet.From {
	case domain.PostRequestSetFromResponseBody:
		c.setPreviewFromResponse(id, responseData, fromKey)
	case domain.PostRequestSetFromResponseHeader:
		c.setPreviewFromHeader(id, responseData, fromKey)
	case domain.PostRequestSetFromResponseCookie:
		c.setPreviewFromCookie(id, responseData, fromKey)
	}
}

func (c *Controller) setPreviewFromResponse(id string, responseData *domain.HTTPResponseDetail, fromKey string) {
	resp, err := rest.GetJSONPATH(responseData.Response, fromKey)
	if err != nil {
		// TODO show error without interrupting the user
		// c.view.showError(fmt.Errorf("failed to get data from response, %w", err))
		return
	}

	if resp == nil {
		return
	}

	if result, ok := resp.(string); ok {
		c.view.SetPostRequestSetPreview(id, result)
	}
}

func (c *Controller) setPreviewFromHeader(id string, responseData *domain.HTTPResponseDetail, fromKey string) {
	for _, header := range responseData.Headers {
		if header.Key == fromKey {
			c.view.SetPostRequestSetPreview(id, header.Value)
			return
		}
	}
}

func (c *Controller) setPreviewFromCookie(id string, responseData *domain.HTTPResponseDetail, fromKey string) {
	for _, cookie := range responseData.Cookies {
		if cookie.Key == fromKey {
			c.view.SetPostRequestSetPreview(id, cookie.Value)
			return
		}
	}
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

func (c *Controller) onCopyResponse(gtx layout.Context, dataType, data string) {
	gtx.Execute(clipboard.WriteCmd{
		Data: io.NopCloser(strings.NewReader(data)),
	})

	c.view.showNotification(fmt.Sprintf("%s copied to clipboard", dataType), 2*time.Second)

	// notify.Send(fmt.Sprintf("%s copied to clipboard", dataType), 2*time.Second)
}

func (c *Controller) onSubmitRequest(id string) {
	c.view.SetSendingRequestLoading(id)
	defer c.view.SetSendingRequestLoaded(id)

	res, err := c.restService.SendRequest(id, c.getActiveEnvID())
	if err != nil {
		c.view.SetHTTPResponse(id, domain.HTTPResponseDetail{
			Error: err,
		})
		return
	}

	resp := string(res.Body)
	if res.IsJSON {
		resp = res.JSON
	}

	c.view.SetHTTPResponse(id, domain.HTTPResponseDetail{
		Response:   resp,
		Headers:    mapToKeyValue(res.Headers),
		Cookies:    cookieToKeyValue(res.Cookies),
		StatusCode: res.StatusCode,
		Duration:   res.TimePassed,
		Size:       len(res.Body),
	})
}

func cookieToKeyValue(cookies []*http.Cookie) []domain.KeyValue {
	var kvs = make([]domain.KeyValue, 0, len(cookies))
	for _, c := range cookies {
		kvs = append(kvs, domain.KeyValue{Key: c.Name, Value: c.Value})
	}
	return kvs
}

func mapToKeyValue(m map[string]string) []domain.KeyValue {
	var kvs = make([]domain.KeyValue, 0, len(m))
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
		c.view.showError(fmt.Errorf("failed to get request, %s", id))
		return
	}

	inComingRequest, ok := data.(*domain.Request)
	if !ok {
		panic("failed to convert data to Request")
	}

	// is data changed?
	if domain.CompareRequests(req, inComingRequest) {
		return
	}

	c.checkForHTTPRequestParams(req, inComingRequest)

	// break the reference
	clone := inComingRequest.Clone()
	req.Spec = clone.Spec

	if err := c.model.UpdateRequest(req, true); err != nil {
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
		return
	}

	// set tab dirty if the in memory data is different from the file
	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get request from file, %w", err))
		return
	}
	c.view.SetTabDirty(id, !domain.CompareRequests(req, reqFromFile))
	c.view.SetTreeViewNodePrefix(id, req)
}

func (c *Controller) checkForHTTPRequestParams(req *domain.Request, inComingRequest *domain.Request) {
	if req.MetaData.Type != domain.RequestTypeHTTP {
		return
	}

	queryParamsChanged := !domain.CompareKeyValues(req.Spec.HTTP.Request.QueryParams, inComingRequest.Spec.HTTP.Request.QueryParams)
	urlChanged := inComingRequest.Spec.HTTP.URL != req.Spec.HTTP.URL

	// if query params and url are changed, update the url base on the query params
	if (queryParamsChanged && urlChanged) || queryParamsChanged {
		newURL := c.getNewURLWithParams(inComingRequest.Spec.HTTP.Request.QueryParams, inComingRequest.Spec.HTTP.URL)
		c.view.SetURL(req.MetaData.ID, newURL)
		inComingRequest.Spec.HTTP.URL = newURL
	} else if urlChanged {
		// update query params based on the new url
		newParams := c.getUrlParams(inComingRequest.Spec.HTTP.URL)
		c.view.SetQueryParams(req.MetaData.ID, newParams)
		inComingRequest.Spec.HTTP.Request.QueryParams = newParams

		// update the path params based on the new url
		newPathParams := domain.ParsePathParams(inComingRequest.Spec.HTTP.URL)
		c.view.SetPathParams(req.MetaData.ID, newPathParams)
		inComingRequest.Spec.HTTP.Request.PathParams = newPathParams
	}
}

func (c *Controller) getNewURLWithParams(params []domain.KeyValue, url string) string {
	if len(params) == 0 {
		// remove any query params from the url and return it
		if strings.Contains(url, "?") {
			url = strings.Split(url, "?")[0]
		}
		return url
	}

	attach := func(url, params string) string {
		if len(params) == 0 {
			return url
		}

		return url + "?" + params
	}

	// sync url params with url params editor
	if url == "" {
		return ""
	}

	urlParams := strings.Split(url, "?")
	if len(urlParams) < 2 {
		return attach(url, domain.EncodeQueryParams(params))
	}

	return attach(urlParams[0], domain.EncodeQueryParams(params))
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
		c.view.showError(fmt.Errorf("failed to get collection, %s", id))
		return
	}
}

func (c *Controller) onRequestTabClose(id string) {
	// is tab data changed?
	// if yes show prompt
	// if no close tab
	req := c.model.GetRequest(id)
	if req == nil {
		c.view.showError(fmt.Errorf("failed to get request, %s", id))
		return
	}

	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get environment from file, %w", err))
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
			c.view.SetTreeViewNodePrefix(id, reqFromFile)
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
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
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
		c.view.showError(fmt.Errorf("failed to update collection, %w", err))
		return
	}

	c.view.UpdateTreeNodeTitle(col.MetaData.ID, col.MetaData.Name)
	c.view.UpdateTabTitle(col.MetaData.ID, col.MetaData.Name)
}

func (c *Controller) onNewRequest(requestType string) {
	var req *domain.Request
	if requestType == domain.RequestTypeHTTP {
		req = domain.NewHTTPRequest("New Request")
	} else {
		req = domain.NewGRPCRequest("New Request")
	}

	newFilePath, err := c.repo.GetNewRequestFilePath(req.MetaData.Name)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get new file path, %w", err))
		return
	}

	req.FilePath = newFilePath.Path
	req.MetaData.Name = newFilePath.NewName

	c.model.AddRequest(req)
	c.view.AddRequestTreeViewNode(req)
	c.saveRequestToDisc(req.MetaData.ID)
	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	clone, _ := domain.Clone[domain.Request](req)
	clone.MetaData.ID = req.MetaData.ID
	c.view.OpenRequestContainer(clone)
	c.view.SwitchToTab(req.MetaData.ID)
}

func (c *Controller) onImport() {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if !errors.Is(result.Error, explorer.ErrUserDecline) {
			c.view.showError(fmt.Errorf("failed to get file, %w", result.Error))
			return
		}

		if err := importer.ImportPostmanCollection(result.Data); err != nil {
			c.view.showError(fmt.Errorf("failed to import postman collection, %w", err))
			return
		}

		if err := c.LoadData(); err != nil {
			c.view.showError(fmt.Errorf("failed to load collections, %w", err))
			return
		}

	}, "json")
}

func (c *Controller) onNewCollection() {
	col := domain.NewCollection("New Collection")

	dirPath, err := c.repo.GetNewCollectionDir(col.MetaData.Name)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get new collection dir, %w", err))
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
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
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
		c.view.showError(fmt.Errorf("failed to update collection, %w", err))
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
		if nodeType == TypeCollection {
			c.duplicateCollection(id)
		} else {
			c.duplicateRequest(id)
		}
	case MenuDelete:
		switch nodeType {
		case TypeRequest:
			c.deleteRequest(id)
		case TypeCollection:
			c.deleteCollection(id)
		}
	case MenuAddHTTPRequest, MenuAddGRPCRequest:
		requestType := domain.RequestTypeHTTP
		if action == MenuAddGRPCRequest {
			requestType = domain.RequestTypeGRPC
		}

		c.addRequestToCollection(id, requestType)
	case MenuView:
		if nodeType == TypeCollection {
			c.viewCollection(id)
		} else {
			c.viewRequest(id)
		}
	}
}

func (c *Controller) addRequestToCollection(id string, requestType string) {
	var req *domain.Request
	if requestType == domain.RequestTypeHTTP {
		req = domain.NewHTTPRequest("New Request")
	} else {
		req = domain.NewGRPCRequest("New Request")
	}

	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	newFilePath, err := c.repo.GetCollectionRequestNewFilePath(col, req.MetaData.Name)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get new file path, err %w", err))
		return
	}

	req.FilePath = newFilePath.Path
	req.MetaData.Name = newFilePath.NewName
	req.CollectionID = col.MetaData.ID
	req.CollectionName = col.MetaData.Name

	c.model.AddRequest(req)
	c.view.AddChildTreeViewNode(col.MetaData.ID, req)
	c.saveRequestToDisc(req.MetaData.ID)
	c.model.AddRequestToCollection(col, req)
	c.view.ExpandTreeViewNode(col.MetaData.ID)
	clone, _ := domain.Clone[domain.Request](req)
	clone.MetaData.ID = req.MetaData.ID
	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	c.view.OpenRequestContainer(clone)
	c.view.SwitchToTab(req.MetaData.ID)
}

func (c *Controller) viewRequest(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		c.view.showError(fmt.Errorf("failed to get request, %s", id))
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

func (c *Controller) duplicateCollection(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	colClone := col.Clone()
	colClone.MetaData.Name += " (copy)"

	dirPath, err := c.repo.GetNewCollectionDir(colClone.MetaData.Name)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get new collection dir, %w", err))
		return
	}

	colClone.FilePath = dirPath.Path
	colClone.MetaData.Name = dirPath.NewName
	c.model.AddCollection(colClone)
	c.view.AddCollectionTreeViewNode(colClone)
	requests := colClone.Spec.Requests
	colClone.Spec.Requests = nil
	c.saveCollectionToDisc(colClone.MetaData.ID)
	colClone.Spec.Requests = requests

	for _, req := range requests {
		reqClone := req.Clone()

		newFilePath, err := c.repo.GetCollectionRequestNewFilePath(colClone, reqClone.MetaData.Name)
		if err != nil {
			c.view.showError(fmt.Errorf("failed to get new file path, err %w", err))
			return
		}

		reqClone.FilePath = newFilePath.Path
		reqClone.MetaData.Name = newFilePath.NewName
		reqClone.CollectionID = colClone.MetaData.ID
		reqClone.CollectionName = colClone.MetaData.Name
		c.model.AddRequest(reqClone)
		c.view.AddChildTreeViewNode(colClone.MetaData.ID, reqClone)
		c.saveRequestToDisc(reqClone.MetaData.ID)
	}
}

func (c *Controller) duplicateRequest(id string) {
	// read request from file to make sure we have the latest persisted data
	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get request from file, %w", err))
		return
	}

	newReq := reqFromFile.Clone()
	newReq.MetaData.Name += " (copy)"
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
		c.view.showError(fmt.Errorf("failed to remove request, %w", err))
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
		c.view.showError(fmt.Errorf("failed to remove collection, %w", err))
		return
	}
	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
}
