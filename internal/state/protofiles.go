package state

import (
	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/safemap"
)

type (
	ProtoFileChangeListener func(request *domain.ProtoFile, action Action)
)

type ProtoFiles struct {
	protoFilesChangeListeners []ProtoFileChangeListener

	protoFiles *safemap.Map[*domain.ProtoFile]
	repository repository.RepositoryV2
}

func NewProtoFiles(repository repository.RepositoryV2) *ProtoFiles {
	return &ProtoFiles{
		repository: repository,
		protoFiles: safemap.New[*domain.ProtoFile](),
	}
}

func (m *ProtoFiles) AddProtoFileChangeListener(listener ProtoFileChangeListener) {
	m.protoFilesChangeListeners = append(m.protoFilesChangeListeners, listener)
}

func (m *ProtoFiles) notifyProtoFileChange(proto *domain.ProtoFile, action Action) {
	for _, listener := range m.protoFilesChangeListeners {
		listener(proto, action)
	}
}

func (m *ProtoFiles) AddProtoFile(proto *domain.ProtoFile) {
	m.protoFiles.Set(proto.MetaData.ID, proto)
	m.notifyProtoFileChange(proto, ActionAdd)
}

func (m *ProtoFiles) GetProtoFile(id string) *domain.ProtoFile {
	pr, _ := m.protoFiles.Get(id)
	return pr
}

func (m *ProtoFiles) UpdateProtoFile(proto *domain.ProtoFile, stateOnly bool) error {
	if _, ok := m.protoFiles.Get(proto.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.UpdateProtoFile(proto); err != nil {
			return err
		}
	}

	m.protoFiles.Set(proto.MetaData.ID, proto)
	m.notifyProtoFileChange(proto, ActionUpdate)

	return nil
}

func (m *ProtoFiles) RemoveProtoFile(proto *domain.ProtoFile, stateOnly bool) error {
	if _, ok := m.protoFiles.Get(proto.MetaData.ID); !ok {
		return ErrNotFound
	}

	if !stateOnly {
		if err := m.repository.DeleteProtoFile(proto); err != nil {
			return err
		}
	}

	m.protoFiles.Delete(proto.MetaData.ID)
	m.notifyProtoFileChange(proto, ActionDelete)

	return nil
}

func (m *ProtoFiles) GetProtoFiles() []*domain.ProtoFile {
	return m.protoFiles.Values()
}

func (m *ProtoFiles) LoadProtoFiles() ([]*domain.ProtoFile, error) {
	protos, err := m.repository.LoadProtoFiles()
	if err != nil {
		return nil, err
	}

	for _, pr := range protos {
		m.protoFiles.Set(pr.MetaData.ID, pr)
	}

	return protos, nil
}
