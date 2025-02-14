package protofiles

import (
	"path/filepath"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/grpc"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/state"
	"github.com/chapar-rest/chapar/ui/explorer"
	"github.com/chapar-rest/chapar/ui/widgets"
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
	view.SetOnAddImportPath(c.addPath)

	return c
}

func (c *Controller) LoadData() error {
	data, err := c.state.LoadProtoFiles()
	if err != nil {
		return err
	}

	c.view.SetItems(data)
	return nil
}

func (c *Controller) showError(title, message string) {
	c.view.ShowPrompt(title, message, widgets.ModalTypeErr, func(selectedOption string, remember bool) {
		if selectedOption == "Ok" {
			c.view.HidePrompt()
			return
		}
	}, []widgets.Option{{Text: "Ok"}}...)
}

func (c *Controller) addPath(path string) {
	// get last part of the path as the name
	fileName := filepath.Base(path)
	proto := domain.NewProtoFile(fileName)

	proto.Spec.IsImportPath = true
	proto.Spec.Path = path

	// Let the repository handle the creation details
	if err := c.repo.Create(proto); err != nil {
		c.showError("Failed to create proto file", err.Error())
		return
	}

	c.state.AddProtoFile(proto)
	c.view.AddItem(proto)
}

func (c *Controller) onAdd() {
	c.explorer.ChoseFile(func(result explorer.Result) {
		if result.Declined {
			return
		}

		if result.Error != nil {
			c.showError("Failed to open file", result.Error.Error())
			return
		}

		fileName := filepath.Base(result.FilePath)
		fileDir := filepath.Dir(result.FilePath)
		proto := domain.NewProtoFile(fileName)

		pInfo, err := c.getProtoInfo(fileDir, fileName)
		if err != nil {
			c.showError("Failed to get proto info", err.Error())
			return
		}

		proto.Spec.Path = result.FilePath
		proto.Spec.Package = pInfo.Package
		proto.Spec.Services = pInfo.Services

		// Let the repository handle the creation details
		if err := c.repo.Create(proto); err != nil {
			c.showError("Failed to create proto file", err.Error())
			return
		}

		c.state.AddProtoFile(proto)
		c.view.AddItem(proto)
	}, "proto")
}

type info struct {
	Package  string
	Services []string
}

func (c *Controller) getProtoInfo(path, filename string) (*info, error) {
	protoFiles, err := c.state.LoadProtoFiles()
	if err != nil {
		return nil, err
	}

	pInfo, err := grpc.ProtoFilesFromDisk(grpc.GetImportPaths(protoFiles, []string{filepath.Join(path, filename)}))
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
		c.showError("Failed to get proto-file", "failed to get proto-file")
		return
	}

	if err := c.state.RemoveProtoFile(pr, false); err != nil {
		c.showError("Failed to remove proto-file", err.Error())
		return
	}

	c.view.RemoveItem(p)
}

func (c *Controller) onDeleteSelected(ids []string) {
	for _, id := range ids {
		pr := c.state.GetProtoFile(id)
		if pr == nil {
			c.showError("Failed to get proto-file", "failed to get proto-file")
			continue
		}

		if err := c.state.RemoveProtoFile(pr, false); err != nil {
			c.showError("Failed to remove proto-file", err.Error())
			continue
		}

		c.view.RemoveItem(pr)
	}
}

func (c *Controller) saveProtoFile(id string) {
	ws := c.state.GetProtoFile(id)
	if ws == nil {
		c.showError("Failed to get proto-file", "failed to get proto-file")
		return
	}

	if err := c.state.UpdateProtoFile(ws, false); err != nil {
		c.showError("Failed to update proto-file", err.Error())
		return
	}
}
