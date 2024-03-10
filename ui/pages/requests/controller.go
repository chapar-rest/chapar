package requests

import (
	"fmt"

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
	view.SetOnNewRequest(c.onNewCollection)
	view.SetOnTitleChanged(c.onTitleChanged)
	view.SetOnTreeViewNodeDoubleClicked(c.onTreeViewNodeDoubleClicked)

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
	case ContainerTypeRequest:
		c.onRequestTitleChange(id, title)
	case ContainerTypeCollection:
		c.onCollectionTitleChange(id, title)
	}
}

func (c *Controller) onRequestTitleChange(id, title string) {
	req := c.model.GetRequest(id)
	if req == nil {
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
}

func (c *Controller) onNewCollection() {
	col := domain.NewCollection("New Collection")
	c.model.AddCollection(col)
	c.view.AddCollectionTreeViewNode(col)
	c.saveCollectionToDisc(col.MetaData.ID)
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

func (c *Controller) onTreeViewNodeDoubleClicked(id string) {
	req := c.model.GetRequest(id)
	if req == nil {
		return
	}

	if c.view.IsTabOpen(id) {
		c.view.SwitchToTab(req.MetaData.ID)
		c.view.OpenRequestContainer(req)
		return
	}

	c.view.OpenRequestTab(req)
	c.view.OpenRequestContainer(req)
}
