package requests

import (
	"fmt"

	"github.com/chapar-rest/chapar/internal/domain"
)

func (v *BaseView) CreateNew(key string) {
	switch key {
	case domain.RequestTypeGRPC:
		v.createNewGRPCRequest()
	case domain.RequestTypeHTTP:
		v.createNewHTTPRequest()
	case domain.KindCollection:
		v.createNewCollection()
	}
}

func (v *BaseView) createNewCollection() {
	col := domain.NewCollection("New Collection")
	if err := v.Repository.CreateCollection(col); err != nil {
		v.ShowError(fmt.Errorf("failed to create collection: %w", err))
		return
	}

	v.AddCollectionTreeViewNode(col)
	v.OpenTab(col.MetaData.ID, col.MetaData.Name, TypeCollection)
	v.OpenCollectionContainer(col)
	v.tabHeader.SetSelectedByID(col.MetaData.ID)
}

func (v *BaseView) createNewGRPCRequest() {
	req := domain.NewGRPCRequest("New Request")

	if err := v.Repository.CreateRequest(req, nil); err != nil {
		v.ShowError(fmt.Errorf("failed to create request: %w", err))
		return
	}

	v.AddRequestTreeViewNode(req)
	v.OpenTab(req.MetaData.ID, req.MetaData.Name, TypeRequest)
	v.OpenRequestContainer(req)
	v.tabHeader.SetSelectedByID(req.MetaData.ID)

}

func (v *BaseView) createNewHTTPRequest() {

}
