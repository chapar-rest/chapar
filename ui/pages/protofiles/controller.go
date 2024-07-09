package protofiles

import (
	"fmt"
	"path/filepath"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/grpc"
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
		fileDir := filepath.Dir(result.FilePath)
		proto := domain.NewProtoFile(fileName)
		filePath, err := c.repo.GetNewProtoFilePath(proto.MetaData.Name)
		if err != nil {
			fmt.Println("failed to get new proto file path", err)
			return
		}

		pInfo, err := c.getProtoInfo(fileDir, fileName)
		if err != nil {
			fmt.Println("failed to get proto info", err)
			return
		}

		proto.FilePath = filePath.Path
		proto.MetaData.Name = filePath.NewName
		proto.Spec.Path = result.FilePath
		proto.Spec.Package = pInfo.Package
		proto.Spec.Services = pInfo.Services

		c.state.AddProtoFile(proto)
		c.saveProtoFileToDisc(proto.MetaData.ID)
		c.view.AddItem(proto)
	}, "proto")
}

type info struct {
	Package  string
	Services []string
}

func (c *Controller) getProtoInfo(path, filename string) (*info, error) {
	pInfo, err := grpc.ProtoFilesFromDisk([]string{path}, []string{filename})
	if err != nil {
		return nil, err
	}

	out := &info{}

	pInfo.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		out.Package = string(f.Package())
		for i := 0; i < f.Services().Len(); i++ {
			out.Services = append(out.Services, string(f.Services().Get(i).FullName()))
		}
		return true
	})

	return out, nil
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
