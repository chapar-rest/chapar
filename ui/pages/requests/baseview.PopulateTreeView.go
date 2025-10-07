package requests

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/widgets"
)

func (v *BaseView) PopulateTreeView(requests []*domain.Request, collections []*domain.Collection) {
	treeViewNodes := make([]*widgets.TreeNode, 0)
	for _, cl := range collections {
		parentNode := &widgets.TreeNode{
			Text:        cl.MetaData.Name,
			Identifier:  cl.MetaData.ID,
			Children:    make([]*widgets.TreeNode, 0),
			MenuOptions: []string{MenuAddHTTPRequest, MenuAddGRPCRequest, MenuDuplicate, MenuView, MenuDelete},
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
		}

		treeViewNodes = append(treeViewNodes, parentNode)
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
	}

	v.treeView.SetNodes(treeViewNodes)
}
