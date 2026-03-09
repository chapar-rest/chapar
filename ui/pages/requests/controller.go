package requests

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/layout"

	"github.com/google/uuid"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/egress"
	"github.com/chapar-rest/chapar/internal/egress/grpc"
	"github.com/chapar-rest/chapar/internal/importer"
	"github.com/chapar-rest/chapar/internal/jsonpath"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/modals"
	"github.com/chapar-rest/chapar/ui/notifications"
	"github.com/chapar-rest/chapar/ui/widgets"
)

type Controller struct {
	model *state.Requests
	view  *View

	envState *state.Environments

	repo repository.RepositoryV2

	explorer *explorer.Explorer

	grpcService   *grpc.Service
	egressService *egress.Service
}

func NewController(view *View, repo repository.RepositoryV2, model *state.Requests, envState *state.Environments, explorer *explorer.Explorer, egressService *egress.Service, grpcService *grpc.Service) *Controller {
	c := &Controller{
		view:     view,
		model:    model,
		repo:     repo,
		envState: envState,

		explorer: explorer,

		egressService: egressService,
		grpcService:   grpcService,
	}

	view.SetController(c)
	return c
}

func (c *Controller) LoadData() error {
	collections, err := c.model.LoadCollections()
	if err != nil {
		return err
	}

	requests, err := c.model.LoadRequests()
	if err != nil {
		return err
	}

	c.view.PopulateTreeView(requests, collections)
	return nil
}

func (c *Controller) OnBinaryFileSelect(id string) {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Declined {
			return
		}

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

func (c *Controller) OnFormDataFileSelect(requestId, fieldId string) {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Declined {
			return
		}

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

func (c *Controller) OnServerInfoReload(id string) {
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

func (c *Controller) OnCreateCollectionFromMethods(requestID string) {
	req := c.model.GetRequest(requestID)
	if req == nil || req.Spec.GRPC == nil {
		return
	}
	methodCount := 0
	for _, s := range req.Spec.GRPC.Services {
		methodCount += len(s.Methods)
	}
	if methodCount == 0 {
		c.view.showError(fmt.Errorf("no methods loaded; load methods via reflection or proto file first"))
		return
	}
	m := modals.NewInputText("Collection name", "Enter collection name")
	c.view.SetModal(func(gtx layout.Context) layout.Dimensions {
		if m.CloseBtn.Clicked(gtx) {
			c.view.CloseModal()
		}
		if m.AddBtn.Clicked(gtx) {
			name := strings.TrimSpace(m.TextField.GetText())
			if name == "" {
				name = "New Collection"
			}
			c.doCreateCollectionFromGRPCMethods(requestID, name)
			c.view.CloseModal()
		}
		return m.Layout(gtx, c.view.Theme)
	})
}

func (c *Controller) doCreateCollectionFromGRPCMethods(requestID, collectionName string) {
	req := c.model.GetRequest(requestID)
	if req == nil || req.Spec.GRPC == nil {
		return
	}
	col := domain.NewCollection(collectionName)
	if err := c.repo.CreateCollection(col); err != nil {
		c.view.showError(fmt.Errorf("failed to create collection: %w", err))
		return
	}
	c.model.AddCollection(col)
	c.view.AddCollectionTreeViewNode(col)

	spec := req.Spec.GRPC
	for _, service := range spec.Services {
		for _, method := range service.Methods {
			requestName := method.Name
			if len(spec.Services) > 1 {
				requestName = service.Name + "." + method.Name
			}
			newReq := &domain.Request{
				ApiVersion: domain.ApiVersion,
				Kind:       domain.KindRequest,
				MetaData: domain.RequestMeta{
					ID:   uuid.NewString(),
					Name: requestName,
					Type: domain.RequestTypeGRPC,
				},
				Spec: domain.RequestSpec{
					GRPC: &domain.GRPCRequestSpec{
						LasSelectedMethod: method.FullName,
						Metadata:          spec.Metadata,
						Auth:              spec.Auth,
						ServerInfo:        spec.ServerInfo,
						Settings:          spec.Settings,
						Body:              "{}",
						Services:          spec.Services,
						Variables:         spec.Variables,
						PreRequest:        spec.PreRequest,
						PostRequest:       spec.PostRequest,
					},
				},
				CollectionID:   col.MetaData.ID,
				CollectionName: col.MetaData.Name,
			}
			newReq.SetDefaultValues()
			if err := c.repo.CreateRequest(newReq, col); err != nil {
				c.view.showError(fmt.Errorf("failed to create request %s: %w", requestName, err))
				continue
			}
			c.model.AddRequest(newReq)
			c.model.AddRequestToCollection(col, newReq)
			c.view.AddChildTreeViewNode(col.MetaData.ID, newReq)
		}
	}
	c.view.ExpandTreeViewNode(col.MetaData.ID)
	c.view.OpenTab(col.MetaData.ID, col.MetaData.Name, TypeCollection)
	c.view.OpenCollectionContainer(col)
	c.view.SwitchToTab(col.MetaData.ID)
}

func (c *Controller) OnGrpcInvoke(id string) {
	c.view.SetSendingRequestLoading(id)
	defer c.view.SetSendingRequestLoaded(id)

	res, err := c.egressService.Send(id, c.getActiveEnvID())
	if err != nil {
		c.view.SetGRPCResponse(id, domain.GRPCResponseDetail{
			Error: err,
		})
		return
	}

	resp, ok := res.(*egress.Response)
	if !ok {
		panic("invalid response type")
	}

	c.view.SetGRPCResponse(id, domain.GRPCResponseDetail{
		Response:         string(resp.Body),
		ResponseMetadata: resp.ResponseMetadata,
		RequestMetadata:  resp.RequestMetadata,
		Trailers:         resp.Trailers,
		StatusCode:       resp.StatueCode,
		Duration:         resp.TimePassed,
		Status:           resp.Status,
		Size:             resp.Size,
	})
}

func (c *Controller) OnGrpcLoadRequestExample(id string) {
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

func (c *Controller) OnPostRequestSetChanged(id string, statusCode int, item, from, fromKey string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	// break the reference
	clone := req.Clone()
	clone.MetaData.ID = id

	// Initialize PostRequestSet
	postRequestSet := domain.PostRequestSet{
		Target:     item,
		StatusCode: statusCode,
		From:       from,
		FromKey:    fromKey,
	}

	// Assign the PostRequestSet based on request type
	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		clone.Spec.HTTP.Request.PostRequest.PostRequestSet = postRequestSet
	case domain.RequestTypeGRPC:
		clone.Spec.GRPC.PostRequest.PostRequestSet = postRequestSet
	case domain.RequestTypeGraphQL:
		clone.Spec.GraphQL.PostRequest.PostRequestSet = postRequestSet
	default:
		return // Unknown request type, exit early
	}

	// Update the request data
	c.onRequestDataChanged(id, clone)

	var (
		responseFrom string
		response     string
		headers      []domain.KeyValue
		cookies      []domain.KeyValue
		metaData     []domain.KeyValue
		trailers     []domain.KeyValue
	)

	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		responseData := c.view.GetHTTPResponse(id)
		if responseData == nil || responseData.Response == "" {
			return
		}
		response = responseData.Response
		headers = responseData.ResponseHeaders
		cookies = responseData.Cookies
		responseFrom = clone.Spec.HTTP.Request.PostRequest.PostRequestSet.From
	case domain.RequestTypeGRPC:
		responseData := c.view.GetGRPCResponse(id)
		if responseData == nil || responseData.Response == "" {
			return
		}
		response = responseData.Response
		headers = responseData.ResponseMetadata
		metaData = responseData.ResponseMetadata
		trailers = responseData.Trailers
		responseFrom = clone.Spec.GRPC.PostRequest.PostRequestSet.From
	case domain.RequestTypeGraphQL:
		responseData := c.view.GetGraphQLResponse(id)
		if responseData == nil || responseData.Response == "" {
			return
		}
		response = responseData.Response
		headers = responseData.ResponseHeaders
		responseFrom = clone.Spec.GraphQL.PostRequest.PostRequestSet.From
	}

	switch responseFrom {
	case domain.PostRequestSetFromResponseBody:
		c.setPreviewFromResponse(id, response, fromKey)
	case domain.PostRequestSetFromResponseHeader:
		c.setPreviewFromKeyValue(id, headers, fromKey)
	case domain.PostRequestSetFromResponseCookie:
		c.setPreviewFromKeyValue(id, cookies, fromKey)
	case domain.PostRequestSetFromResponseMetaData:
		c.setPreviewFromKeyValue(id, metaData, fromKey)
	case domain.PostRequestSetFromResponseTrailers:
		c.setPreviewFromKeyValue(id, trailers, fromKey)
	}
}

func (c *Controller) OnSetOnTriggerRequestChanged(id, collectionID, requestID string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	// break the reference
	clone := req.Clone()
	clone.MetaData.ID = id

	triggerRequest := &domain.TriggerRequest{
		CollectionID: collectionID,
		RequestID:    requestID,
	}

	// Assign the PostRequestSet based on request type
	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		clone.Spec.HTTP.Request.PreRequest.TriggerRequest = triggerRequest
	case domain.RequestTypeGRPC:
		clone.Spec.GRPC.PreRequest.TriggerRequest = triggerRequest
	case domain.RequestTypeGraphQL:
		clone.Spec.GraphQL.PreRequest.TriggerRequest = triggerRequest
	default:
		return // Unknown request type, exit early
	}

	// Update the request data
	c.onRequestDataChanged(id, clone)
}

func (c *Controller) setPreviewFromResponse(id string, response, fromKey string) {
	resp, err := jsonpath.Get(response, fromKey)
	if err != nil {
		// TODO show error without interrupting the user
		fmt.Println("failed to get data from response, %w", err)
		// c.view.showError(fmt.Errorf("failed to get data from response, %w", err))
		return
	}

	if resp == nil {
		return
	}

	if result, ok := resp.(string); ok {
		c.view.SetPostRequestSetPreview(id, result)
	} else {
		c.view.SetPostRequestSetPreview(id, fmt.Sprintf("%v", resp))
	}
}

func (c *Controller) setPreviewFromKeyValue(id string, kv []domain.KeyValue, fromKey string) {
	for _, header := range kv {
		if header.Key == fromKey {
			c.view.SetPostRequestSetPreview(id, header.Value)
			return
		}
	}
}

func (c *Controller) OnTitleChanged(id string, title, containerType string) {
	switch containerType {
	case TypeRequest:
		c.onRequestTitleChange(id, title)
	case TypeCollection:
		c.onCollectionTitleChange(id, title)
	}
}

func (c *Controller) OnSave(id string) {
	tabType := c.view.GetTabType(id)
	if tabType == TypeRequest {
		c.saveRequest(id)
	} else if tabType == TypeCollection {
		c.saveCollection(id)
	}
}

func (c *Controller) OnSubmit(id, containerType string) {
	if containerType == TypeRequest {
		c.onSubmitRequest(id)
	}
}

func (c *Controller) OnCopyResponse(gtx layout.Context, dataType, data string) {
	gtx.Execute(clipboard.WriteCmd{
		Data: io.NopCloser(strings.NewReader(data)),
	})

	notifications.Send(fmt.Sprintf("%s copied to clipboard", dataType), notifications.NotificationTypeInfo, 2*time.Second)
}

func (c *Controller) onSubmitRequest(id string) {
	c.view.SetSendingRequestLoading(id)
	defer c.view.SetSendingRequestLoaded(id)

	req := c.model.GetRequest(id)
	if req == nil {
		c.view.showError(fmt.Errorf("request with id %s not found", id))
		return
	}

	egRes, err := c.egressService.Send(id, c.getActiveEnvID())
	if err != nil {
		// Handle error based on request type
		switch req.MetaData.Type {
		case domain.RequestTypeHTTP:
			c.view.SetHTTPResponse(id, domain.HTTPResponseDetail{
				Error: err,
			})
		case domain.RequestTypeGraphQL:
			c.view.SetGraphQLResponse(id, domain.GraphQLResponseDetail{
				Error: err,
			})
		case domain.RequestTypeGRPC:
			c.view.SetGRPCResponse(id, domain.GRPCResponseDetail{
				Error: err,
			})
		}
		return
	}

	// Handle response based on request type
	if req.MetaData.Type == domain.RequestTypeHTTP {
		res, ok := egRes.(*egress.Response)
		if !ok {
			panic("invalid response type")
		}

		resp := string(res.Body)
		if res.IsJSON {
			resp = res.JSON
		}

		c.view.SetHTTPResponse(id, domain.HTTPResponseDetail{
			Response:        resp,
			ResponseHeaders: mapToKeyValue(res.ResponseHeaders),
			RequestHeaders:  mapToKeyValue(res.RequestHeaders),
			Cookies:         cookieToKeyValue(res.Cookies),
			StatusCode:      res.StatusCode,
			Duration:        res.TimePassed,
			Size:            len(res.Body),
		})

		return
	}

	if req.MetaData.Type == domain.RequestTypeGraphQL {
		res, ok := egRes.(*egress.Response)
		if !ok {
			panic("invalid response type")
		}

		resp := string(res.Body)
		if res.IsJSON {
			resp = res.JSON
		}

		c.view.SetGraphQLResponse(id, domain.GraphQLResponseDetail{
			Response:        resp,
			ResponseHeaders: mapToKeyValue(res.ResponseHeaders),
			RequestHeaders:  mapToKeyValue(res.RequestHeaders),
			StatusCode:      res.StatusCode,
			Duration:        res.TimePassed,
			Size:            len(res.Body),
		})
	}
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

func (c *Controller) OnTabClose(id string) {
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

func (c *Controller) OnDataChanged(id string, data any, containerType string) {
	switch containerType {
	case TypeRequest:
		c.onRequestDataChanged(id, data)
	case TypeCollection:
		c.onCollectionDataChanged(id, data)
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

	c.checkForPreRequestParams(id, req, inComingRequest)
	c.checkForHTTPRequestParams(req, inComingRequest)

	// break the reference
	clone := inComingRequest.Clone()
	req.Spec = clone.Spec

	if err := c.model.UpdateRequest(req); err != nil {
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
		return
	}

	// set tab dirty if the in memory data is different from the file
	reqFromFile, err := c.model.GetPersistedRequest(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get request from file, %w", err))
		return
	}
	c.view.SetTabDirty(id, !domain.CompareRequests(req, reqFromFile))
	c.view.SetTreeViewNodePrefix(id, req)
}

func (c *Controller) checkForPreRequestParams(id string, req *domain.Request, inComingRequest *domain.Request) {
	var (
		reqType        domain.RequestType
		preReq         *domain.PreRequest
		incomingPreReq *domain.PreRequest
	)

	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		reqType = domain.RequestTypeHTTP
		preReq = &req.Spec.HTTP.Request.PreRequest
		incomingPreReq = &inComingRequest.Spec.HTTP.Request.PreRequest
	case domain.RequestTypeGRPC:
		reqType = domain.RequestTypeGRPC
		preReq = &req.Spec.GRPC.PreRequest
		incomingPreReq = &inComingRequest.Spec.GRPC.PreRequest
	}

	if reqType != "" && (preReq.Type != incomingPreReq.Type || incomingPreReq.Type == domain.PrePostTypeTriggerRequest) {
		var (
			collectionID = domain.PrePostTypeNone
			requestID    string
		)

		if incomingPreReq.TriggerRequest != nil {
			collectionID = incomingPreReq.TriggerRequest.CollectionID
			requestID = incomingPreReq.TriggerRequest.RequestID
		}

		c.view.SetPreRequestCollections(id, c.model.GetCollections(), collectionID)
		if collectionID != domain.PrePostTypeNone {
			requests := c.model.GetCollection(collectionID).Spec.Requests
			c.view.SetPreRequestRequests(id, requests, requestID)
		} else {
			c.view.SetPreRequestRequests(id, c.model.GetStandAloneRequests(), requestID)
		}
	}
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
		// keep the values of the path params that are already set
		for _, param := range inComingRequest.Spec.HTTP.Request.PathParams {
			for i, newParam := range newPathParams {
				if newParam.Key == param.Key {
					newPathParams[i].Value = param.Value
				}
			}
		}

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

func (c *Controller) onCollectionDataChanged(id string, data any) {
	col := c.model.GetCollection(id)
	if col == nil {
		c.view.showError(fmt.Errorf("failed to get collection, %s", id))
		return
	}

	incomingCollection, ok := data.(*domain.Collection)
	if !ok {
		panic("failed to convert data to Collection")
	}

	// Update headers and auth from incoming data
	col.Spec.Headers = incomingCollection.Spec.Headers
	col.Spec.Auth = incomingCollection.Spec.Auth

	// Update the collection in the model
	if err := c.model.UpdateCollection(col, true); err != nil {
		c.view.showError(fmt.Errorf("failed to update collection, %w", err))
		return
	}

	// Set tab dirty if the in memory data is different from the file
	collections, err := c.model.LoadCollections()
	if err != nil {
		c.view.showError(fmt.Errorf("failed to load collections from file, %w", err))
		return
	}

	var colFromFile *domain.Collection
	for _, c := range collections {
		if c.MetaData.ID == id {
			colFromFile = c
			break
		}
	}

	if colFromFile != nil {
		// Simple comparison: check if headers or auth changed
		headersChanged := !domain.CompareKeyValues(col.Spec.Headers, colFromFile.Spec.Headers)
		authChanged := !domain.CompareAuth(col.Spec.Auth, colFromFile.Spec.Auth)
		c.view.SetTabDirty(id, headersChanged || authChanged)
	} else {
		c.view.SetTabDirty(id, true)
	}

	if err := c.model.UpdateCollection(col, true); err != nil {
		c.view.showError(fmt.Errorf("failed to update collection, %w", err))
		return
	}

	// Update all open request containers that belong to this collection
	c.view.UpdateCollectionForOpenRequests(id, col, func(requestID string) bool {
		req := c.model.GetRequest(requestID)
		return req != nil && req.CollectionID == id
	})
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

	reqFromFile, err := c.model.GetPersistedRequest(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get request from file, %w", err))
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
				c.saveRequest(id)
			}

			c.view.CloseTab(id)
			c.model.ReloadRequest(id)
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

	col := c.model.GetCollection(req.CollectionID)
	if err := c.repo.UpdateRequest(req, col); err != nil {
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
		return
	}

	// update the state
	if err := c.model.UpdateRequest(req); err != nil {
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
		return
	}

	c.view.UpdateTreeNodeTitle(req.MetaData.ID, req.MetaData.Name)
	c.view.UpdateTabTitle(req.MetaData.ID, req.MetaData.Name)
	c.view.SetContainerTitle(id, req.MetaData.Name)
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
	c.view.SetContainerTitle(id, col.MetaData.Name)
}

func (c *Controller) OnNewRequest(requestType domain.RequestType) {
	var req *domain.Request
	switch requestType {
	case domain.RequestTypeHTTP:
		req = domain.NewHTTPRequest("New Request")
	case domain.RequestTypeGRPC:
		req = domain.NewGRPCRequest("New Request")
	case domain.RequestTypeGraphQL:
		req = domain.NewGraphQLRequest("New Request")
	default:
		return
	}

	// Let the repository handle the creation details
	if err := c.repo.CreateRequest(req, nil); err != nil {
		c.view.showError(fmt.Errorf("failed to create request: %w", err))
		return
	}

	// load back the created request from the repository
	req, err := c.model.GetPersistedRequest(req.MetaData.ID)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get request from file, %w", err))
		return
	}

	c.model.AddRequest(req)
	c.view.AddRequestTreeViewNode(req)
	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	clone, _ := domain.Clone[domain.Request](req)
	clone.MetaData.ID = req.MetaData.ID
	c.view.OpenRequestContainer(clone)
	c.view.SwitchToTab(req.MetaData.ID)
}

func (c *Controller) OnImport(importType string) {
	var fileExtension string
	var importFunc func([]byte, repository.RepositoryV2, ...string) error

	switch importType {
	case "postman":
		fileExtension = "json"
		importFunc = func(data []byte, repo repository.RepositoryV2, _ ...string) error {
			return importer.ImportPostmanCollection(data, repo)
		}
	case "openapi":
		fileExtension = "json,yaml,yml"
		importFunc = func(data []byte, repo repository.RepositoryV2, _ ...string) error {
			return importer.ImportOpenAPISpec(data, repo)
		}
	case "protofile":
		fileExtension = "proto"
		importFunc = importer.ImportProtoFile
	default:
		c.view.showError(fmt.Errorf("unsupported import type: %s", importType))
		return
	}

	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Declined {
			return
		}

		if result.Error != nil {
			c.view.showError(fmt.Errorf("failed to get file, %w", result.Error))
			return
		}

		if err := importFunc(result.Data, c.repo, result.FilePath); err != nil {
			c.view.showError(fmt.Errorf("failed to import %s, %w", importType, err))
			return
		}

		if err := c.LoadData(); err != nil {
			c.view.showError(fmt.Errorf("failed to load collections, %w", err))
			return
		}

	}, fileExtension)
}

func (c *Controller) OnNewCollection() {
	col := domain.NewCollection("New Collection")

	if err := c.repo.CreateCollection(col); err != nil {
		c.view.showError(fmt.Errorf("failed to create collection: %w", err))
		return
	}

	c.model.AddCollection(col)
	c.view.AddCollectionTreeViewNode(col)
	c.view.OpenTab(col.MetaData.ID, col.MetaData.Name, TypeCollection)
	c.view.OpenCollectionContainer(col)
	c.view.SwitchToTab(col.MetaData.ID)
}

func (c *Controller) saveRequest(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	col := c.model.GetCollection(req.CollectionID)
	if err := c.repo.UpdateRequest(req, col); err != nil {
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
		return
	}

	// update the state
	if err := c.model.UpdateRequest(req); err != nil {
		c.view.showError(fmt.Errorf("failed to update request, %w", err))
		return
	}
	c.view.SetTabDirty(id, false)
}

func (c *Controller) saveCollection(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	if err := c.repo.UpdateCollection(col); err != nil {
		c.view.showError(fmt.Errorf("failed to update collection, %w", err))
		return
	}

	// update the state
	if err := c.model.UpdateCollection(col, false); err != nil {
		c.view.showError(fmt.Errorf("failed to update collection, %w", err))
		return
	}
	c.view.SetTabDirty(id, false)
}

func (c *Controller) OnTreeViewNodeClicked(id string) {
	nodeType := c.view.GetTreeViewNodeType(id)
	if nodeType == "" {
		return
	}

	switch nodeType {
	case TypeCollection:
		c.viewCollection(id)
	case TypeRequest:
		c.viewRequest(id)
	}
}

func (c *Controller) OnTreeViewMenuClicked(id, action string) {
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
	case MenuAddHTTPRequest, MenuAddGRPCRequest, MenuAddGraphQLRequest:
		requestType := domain.RequestTypeHTTP
		if action == MenuAddGRPCRequest {
			requestType = domain.RequestTypeGRPC
		} else if action == MenuAddGraphQLRequest {
			requestType = domain.RequestTypeGraphQL
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

func (c *Controller) addRequestToCollection(id string, requestType domain.RequestType) {
	var req *domain.Request
	switch requestType {
	case domain.RequestTypeHTTP:
		req = domain.NewHTTPRequest("New Request")
	case domain.RequestTypeGRPC:
		req = domain.NewGRPCRequest("New Request")
	case domain.RequestTypeGraphQL:
		req = domain.NewGraphQLRequest("New Request")
	default:
		return
	}

	col := c.model.GetCollection(id)
	if col == nil {
		return
	}

	// Let the repository handle the creation details
	if err := c.repo.CreateRequest(req, col); err != nil {
		c.view.showError(fmt.Errorf("failed to create request: %w", err))
		return
	}

	req.CollectionID = col.MetaData.ID
	req.CollectionName = col.MetaData.Name

	c.model.AddRequest(req)
	c.view.AddChildTreeViewNode(col.MetaData.ID, req)
	c.model.AddRequestToCollection(col, req)
	c.view.ExpandTreeViewNode(col.MetaData.ID)
	clone, _ := domain.Clone(req)
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
		// Set collection if request belongs to one
		if req.CollectionID != "" {
			collection := c.model.GetCollection(req.CollectionID)
			if collection != nil {
				c.view.SetRequestCollection(id, collection)
			}
		}
		return
	}

	// make a clone to keep the original request unchanged
	// clone, _ := domain.Clone[domain.Request](req)
	clone := req.Clone()
	clone.MetaData.ID = id

	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	c.view.OpenRequestContainer(clone)

	// Set collection if request belongs to one
	if req.CollectionID != "" {
		collection := c.model.GetCollection(req.CollectionID)
		if collection != nil {
			c.view.SetRequestCollection(id, collection)
		}
	}
}

func (c *Controller) OpenCollection(id string) {
	c.viewCollection(id)
}

func (c *Controller) OpenRequest(id string) {
	c.viewRequest(id)
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

	// Create a clone of the collection with "(copy)" suffix
	colClone := col.Clone()
	// Let the repository handle the collection creation
	if err := c.repo.CreateCollection(colClone); err != nil {
		c.view.showError(fmt.Errorf("failed to create collection: %w", err))
		return
	}

	c.model.AddCollection(colClone)
	c.view.AddCollectionTreeViewNode(colClone)

	// Duplicate each request in the collection
	for _, req := range colClone.Spec.Requests {
		// Let the repository handle the request creation in the collection
		if err := c.repo.CreateRequest(req, colClone); err != nil {
			c.view.showError(fmt.Errorf("failed to create request: %w", err))
			continue
		}

		c.model.AddRequest(req)
		c.view.AddChildTreeViewNode(colClone.MetaData.ID, req)
	}
}

func (c *Controller) duplicateRequest(id string) {
	reqFromFile, err := c.model.GetPersistedRequest(id)
	if err != nil {
		c.view.showError(fmt.Errorf("failed to get request from file, %w", err))
		return
	}

	newReq := reqFromFile.Clone()
	newReq.MetaData.Name += " (copy)"

	// If the request is part of a collection, set the collection ID and name
	col := c.model.GetCollection(reqFromFile.CollectionID)

	// Let the repository handle the creation details
	if err := c.repo.CreateRequest(newReq, col); err != nil {
		c.view.showError(fmt.Errorf("failed to create request: %w", err))
		return
	}

	c.model.AddRequest(newReq)
	if reqFromFile.CollectionID == "" {
		c.view.AddRequestTreeViewNode(newReq)
	} else {
		c.view.AddChildTreeViewNode(reqFromFile.CollectionID, newReq)
	}
}

func (c *Controller) deleteRequest(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	col := c.model.GetCollection(req.CollectionID)
	if err := c.repo.DeleteRequest(req, col); err != nil {
		c.view.showError(fmt.Errorf("failed to delete request, %w", err))
		return
	}

	if err := c.model.RemoveRequest(req); err != nil {
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

	for _, req := range col.Spec.Requests {
		c.view.RemoveTreeViewNode(req.MetaData.ID)
		c.view.CloseTab(req.MetaData.ID)
	}

	if err := c.model.RemoveCollection(col, false); err != nil {
		c.view.showError(fmt.Errorf("failed to remove collection, %w", err))
		return
	}

	c.view.RemoveTreeViewNode(id)
	c.view.CloseTab(id)
}

func (c *Controller) OnRequestTabChanged(id, tab string) {
	if tab != "Pre Request" {
		return
	}

	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	var (
		collectionID = domain.PrePostTypeNone
		requestID    string

		preRequest *domain.PreRequest
	)

	switch req.MetaData.Type {
	case domain.RequestTypeHTTP:
		preRequest = &req.Spec.HTTP.Request.PreRequest
	case domain.RequestTypeGRPC:
		preRequest = &req.Spec.GRPC.PreRequest
	case domain.RequestTypeGraphQL:
		preRequest = &req.Spec.GraphQL.PreRequest
	}

	if preRequest == nil {
		return
	}

	if preRequest.TriggerRequest != nil {
		collectionID = preRequest.TriggerRequest.CollectionID
		requestID = preRequest.TriggerRequest.RequestID
	}

	c.view.SetPreRequestCollections(id, c.model.GetCollections(), collectionID)
	if collectionID != domain.PrePostTypeNone {
		requests := c.model.GetCollection(collectionID).Spec.Requests
		c.view.SetPreRequestRequests(id, requests, requestID)
	} else {
		c.view.SetPreRequestRequests(id, c.model.GetRequests(), requestID)
	}
}
