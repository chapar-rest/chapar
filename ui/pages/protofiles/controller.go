package protofiles

import (
	"fmt"
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/explorer"
)

type Controller struct {
	view *View

	state *state.ProtoFiles
	repo  repository.Repository

	explorer *explorer.Explorer
}

func NewController(view *View, state *state.ProtoFiles, repo repository.Repository, explorer *explorer.Explorer) *Controller {
	c := &Controller{
		view:     view,
		state:    state,
		repo:     repo,
		explorer: explorer,
	}

	view.SetOnAdd(c.onAdd)
	view.SetOnDelete(c.onDelete)
	view.SetOnDeleteSelected(c.onDeleteSelected)

	return c
}

func (c *Controller) LoadData() error {
	data, err := c.state.LoadProtoFilesFromDisk()
	if err != nil {
		return err
	}

	c.view.SetItems(data)
	return nil
}

func (c *Controller) onAdd() {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Error != nil {
			fmt.Println("failed to get file", result.Error)
			return
		}

		fileName := filepath.Base(result.FilePath)
		proto := domain.NewProtoFile(fileName)
		filePath, err := c.repo.GetNewProtoFilePath(proto.MetaData.Name)
		if err != nil {
			fmt.Println("failed to get new proto file path", err)
			return
		}

		proto.FilePath = filePath.Path
		proto.Spec.Path = result.FilePath
		proto.MetaData.Name = filePath.NewName

		c.state.AddProtoFile(proto)
		c.saveProtoFileToDisc(proto.MetaData.ID)
		c.view.AddItem(proto)
	}, "proto")
}

func (c *Controller) onDelete(p *domain.ProtoFile) {
	pr := c.state.GetProtoFile(p.MetaData.ID)
	if pr == nil {
		fmt.Println("failed to get proto-file", p.MetaData.ID)
		return
	}

	if err := c.state.RemoveProtoFile(pr, false); err != nil {
		fmt.Println("failed to remove proto-file", err)
		return
	}

	c.view.RemoveItem(p)
}

func (c *Controller) onDeleteSelected(ids []string) {
	for _, id := range ids {
		pr := c.state.GetProtoFile(id)
		if pr == nil {
			fmt.Println("failed to get proto-file", id)
			continue
		}

		if err := c.state.RemoveProtoFile(pr, false); err != nil {
			fmt.Println("failed to remove proto-file", err)
			continue
		}

		c.view.RemoveItem(pr)
	}
}

func (c *Controller) saveProtoFileToDisc(id string) {
	ws := c.state.GetProtoFile(id)
	if ws == nil {
		fmt.Println("failed to get proto-file", id)
		return
	}

	if err := c.state.UpdateProtoFile(ws, false); err != nil {
		fmt.Println("failed to update proto-file", err)
		return
	}
}
