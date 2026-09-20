package uiv2

import (
	"path/filepath"
	"testing"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

// updateRecorder is a repository that only records the requests written back.
type updateRecorder struct {
	repository.RepositoryV2
	updated []string
}

func (r *updateRecorder) UpdateRequest(req *domain.Request, _ *domain.Collection) error {
	r.updated = append(r.updated, req.MetaData.ID)
	return nil
}

func grpcRequest(id string, protoFiles, importPaths []string) *domain.Request {
	return &domain.Request{
		MetaData: domain.RequestMeta{ID: id, Type: domain.RequestTypeGRPC},
		Spec: domain.RequestSpec{
			GRPC: &domain.GRPCRequestSpec{
				ServerInfo: domain.ServerInfo{
					ProtoFiles:  protoFiles,
					ImportPaths: importPaths,
				},
			},
		},
	}
}

func globals(t *testing.T) (dir string, protos []*domain.ProtoFile) {
	t.Helper()
	dir = t.TempDir()
	file := &domain.ProtoFile{}
	file.Spec.Path = filepath.Join(dir, "vendor", "google", "api", "annotations.proto")
	root := &domain.ProtoFile{}
	root.Spec.Path = filepath.Join(dir, "protos")
	root.Spec.IsImportPath = true
	return dir, []*domain.ProtoFile{file, root}
}

func TestMigrateSeedsImportPathsFromTheOldGlobals(t *testing.T) {
	dir, protos := globals(t)
	req := grpcRequest("r1", []string{filepath.Join(dir, "svc.proto")}, nil)
	repo := &updateRecorder{}

	migrateProtoImportPaths(repo, nil, []*domain.Request{req}, protos)

	want := []string{
		filepath.Join(dir, "vendor", "google", "api"),
		filepath.Join(dir, "protos"),
	}
	got := req.Spec.GRPC.ServerInfo.ImportPaths
	if len(got) != len(want) {
		t.Fatalf("import paths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("import paths = %v, want %v", got, want)
		}
	}
	if len(repo.updated) != 1 || repo.updated[0] != "r1" {
		t.Fatalf("persisted %v, want the migrated request", repo.updated)
	}
}

func TestMigrateLeavesConfiguredRequestsAlone(t *testing.T) {
	dir, protos := globals(t)
	mine := []string{filepath.Join(dir, "mine")}
	req := grpcRequest("r1", []string{filepath.Join(dir, "svc.proto")}, mine)
	repo := &updateRecorder{}

	migrateProtoImportPaths(repo, nil, []*domain.Request{req}, protos)

	if got := req.Spec.GRPC.ServerInfo.ImportPaths; len(got) != 1 || got[0] != mine[0] {
		t.Fatalf("import paths = %v, want the request's own %v", got, mine)
	}
	if len(repo.updated) != 0 {
		t.Fatalf("persisted %v, want nothing rewritten", repo.updated)
	}
}

func TestMigrateSkipsRequestsWithNoProtoFiles(t *testing.T) {
	_, protos := globals(t)
	// A reflection-based request has nothing to parse from disk.
	req := grpcRequest("r1", nil, nil)
	repo := &updateRecorder{}

	migrateProtoImportPaths(repo, nil, []*domain.Request{req}, protos)

	if got := req.Spec.GRPC.ServerInfo.ImportPaths; len(got) != 0 {
		t.Fatalf("import paths = %v, want none", got)
	}
	if len(repo.updated) != 0 {
		t.Fatalf("persisted %v, want nothing rewritten", repo.updated)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	dir, protos := globals(t)
	req := grpcRequest("r1", []string{filepath.Join(dir, "svc.proto")}, nil)
	repo := &updateRecorder{}

	migrateProtoImportPaths(repo, nil, []*domain.Request{req}, protos)
	first := append([]string(nil), req.Spec.GRPC.ServerInfo.ImportPaths...)
	migrateProtoImportPaths(repo, nil, []*domain.Request{req}, protos)

	if got := req.Spec.GRPC.ServerInfo.ImportPaths; len(got) != len(first) {
		t.Fatalf("second run changed import paths: %v then %v", first, got)
	}
	if len(repo.updated) != 1 {
		t.Fatalf("persisted %v, want one write across both runs", repo.updated)
	}
}

func TestMigrateReachesRequestsInsideCollections(t *testing.T) {
	dir, protos := globals(t)
	req := grpcRequest("r1", []string{filepath.Join(dir, "svc.proto")}, nil)
	col := &domain.Collection{}
	col.Spec.Requests = []*domain.Request{req}
	repo := &updateRecorder{}

	migrateProtoImportPaths(repo, []*domain.Collection{col}, nil, protos)

	if len(req.Spec.GRPC.ServerInfo.ImportPaths) == 0 {
		t.Fatal("a request inside a collection was not migrated")
	}
	if len(repo.updated) != 1 {
		t.Fatalf("persisted %v, want the collection's request", repo.updated)
	}
}

func TestMigrateDoesNothingWithoutGlobals(t *testing.T) {
	dir := t.TempDir()
	req := grpcRequest("r1", []string{filepath.Join(dir, "svc.proto")}, nil)
	repo := &updateRecorder{}

	migrateProtoImportPaths(repo, nil, []*domain.Request{req}, nil)

	if got := req.Spec.GRPC.ServerInfo.ImportPaths; len(got) != 0 {
		t.Fatalf("import paths = %v, want none", got)
	}
	if len(repo.updated) != 0 {
		t.Fatalf("persisted %v, want nothing rewritten", repo.updated)
	}
}
