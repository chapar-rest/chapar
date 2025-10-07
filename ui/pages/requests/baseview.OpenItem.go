package requests

import "fmt"

func (v *BaseView) OpenItem(id string) {
	// first check if the tab is already open
	if v.tabHeader.SetSelectedByID(id) {
		return
	}

	// TODO fetch the node and read the meta to decide if it's a request or collection

	col, err := v.Repository.GetCollectionByID(id)
	if err != nil {
		v.ShowError(fmt.Errorf("failed to open item: %w", err))
		return
	}

	v.OpenTab(col.MetaData.ID, col.MetaData.Name, TypeCollection)
	v.OpenCollectionContainer(col)
	v.tabHeader.SetSelectedByID(col.MetaData.ID)
}
