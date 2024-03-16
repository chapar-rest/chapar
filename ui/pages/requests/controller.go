package requests

import (
	"encoding/json"
	"fmt"

	"github.com/mirzakhany/chapar/internal/logger"

	"github.com/mirzakhany/chapar/ui/widgets"

	"github.com/mirzakhany/chapar/internal/domain"
	"github.com/mirzakhany/chapar/internal/loader"
)

type Controller struct {
	model *Model
	view  *View

	activeTabID string
}

func NewController(view *View, model *Model) *Controller {
	c := &Controller{
		view:  view,
		model: model,
	}

	view.SetOnNewRequest(c.onNewRequest)
	view.SetOnNewCollection(c.onNewCollection)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeClicked(c.onTreeViewNodeClicked)
	view.SetOnTreeViewMenuClicked(c.onTreeViewMenuClicked)
	view.SetOnTabClose(c.onTabClose)
	view.SetOnDataChanged(c.onDataChanged)

	return c
}

func (c *Controller) LoadData() error {
	collections, err := loader.LoadCollections()
	if err != nil {
		return err
	}

	requests, err := loader.LoadRequests()
	if err != nil {
		return err
	}

	c.model.SetRequests(requests)
	c.model.SetCollections(collections)
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
		c.onCollectionDataChanged(id, data)
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

	// break the reference
	clone := inComingRequest.Clone()

	req.Spec = clone.Spec
	c.model.UpdateRequest(req)

	// set tab dirty if the in memory data is different from the file
	reqFromFile, err := c.model.GetRequestFromDisc(id)
	if err != nil {
		fmt.Println("failed to get request from file", err)
		return
	}

	prettyPrint(req)
	prettyPrint(reqFromFile)

	cmp := domain.CompareRequests(req, reqFromFile)
	fmt.Println("cmp", cmp)

	c.view.SetTabDirty(id, !cmp)

	fmt.Println("data changed")
}

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

func (c *Controller) onCollectionDataChanged(id string, data any) {
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

	c.view.ShowPrompt(id, "Save", "Do you want to save the changes?", widgets.ModalTypeWarn,
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
		}, "Yes", "No", "Cancel",
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
	c.view.UpdateTreeNodeTitle(req.MetaData.ID, req.MetaData.Name)
	c.view.UpdateTabTitle(req.MetaData.ID, req.MetaData.Name)
	c.model.UpdateRequest(req)
	c.saveRequestToDisc(id)
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
	c.view.UpdateTreeNodeTitle(col.MetaData.ID, col.MetaData.Name)
	c.view.UpdateTabTitle(col.MetaData.ID, col.MetaData.Name)
	c.model.UpdateCollection(col)
	c.saveCollectionToDisc(id)
}

func (c *Controller) onNewRequest() {
	req := domain.NewRequest("New Request")
	c.model.AddRequest(req)
	c.view.AddRequestTreeViewNode(req)
	c.saveRequestToDisc(req.MetaData.ID)
	c.view.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	c.view.OpenRequestContainer(req)
	c.view.SwitchToTab(req.MetaData.ID)
}

func (c *Controller) onNewCollection() {
	col := domain.NewCollection("New Collection")
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
	if err := loader.UpdateRequest(req); err != nil {
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
	if err := loader.UpdateCollection(col); err != nil {
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

	newFilePath, err := loader.GetNewFilePath(req.MetaData.Name, col.MetaData.Name)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get new file path, err %v", err))
		return
	}
	req.FilePath = newFilePath

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
	newReq.FilePath = loader.AddSuffixBeforeExt(newReq.FilePath, "-copy")
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
	c.model.DeleteRequest(id)
	c.view.RemoveTreeViewNode(id)
	if err := loader.DeleteRequest(req); err != nil {
		fmt.Println("failed to delete request", err)
	}
	c.view.CloseTab(id)
}

func (c *Controller) deleteCollection(id string) {
	col := c.model.GetCollection(id)
	if col == nil {
		return
	}
	c.model.DeleteCollection(id)
	c.view.RemoveTreeViewNode(id)
	if err := loader.DeleteCollection(col); err != nil {
		fmt.Println("failed to delete collection", err)
	}
	c.view.CloseTab(id)
}
