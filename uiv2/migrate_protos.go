package uiv2

import (
	"path/filepath"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/repository"
)

// migrateProtoImportPaths moves the workspace-wide proto file list onto the
// gRPC requests that depended on it.
//
// The Proto files page used to feed every entry it held into every gRPC
// parse. Import paths now live on the request, so each request that has proto
// files but no import paths of its own inherits the old global roots once and
// keeps resolving. Requests that already carry import paths are left alone, so
// this is safe to run on every load.
func migrateProtoImportPaths(repo repository.RepositoryV2, cols []*domain.Collection, reqs []*domain.Request, protos []*domain.ProtoFile) {
	roots := protoImportRoots(protos)
	if len(roots) == 0 {
		return
	}

	for _, r := range reqs {
		if seedImportPaths(r, roots) {
			_ = repo.UpdateRequest(r, nil)
		}
	}
	for _, col := range cols {
		for _, r := range col.Spec.Requests {
			if seedImportPaths(r, roots) {
				_ = repo.UpdateRequest(r, col)
			}
		}
	}
}

// seedImportPaths gives a gRPC request the old global import roots, and
// reports whether it changed.
func seedImportPaths(r *domain.Request, roots []string) bool {
	if r == nil || r.Spec.GRPC == nil {
		return false
	}
	info := &r.Spec.GRPC.ServerInfo
	if len(info.ProtoFiles) == 0 || len(info.ImportPaths) > 0 {
		return false
	}
	info.ImportPaths = append([]string(nil), roots...)
	return true
}

// protoImportRoots turns the old proto file documents into import roots: an
// import-path entry is already a root, and a proto file contributes the
// directory that holds it.
func protoImportRoots(protos []*domain.ProtoFile) []string {
	out := make([]string, 0, len(protos))
	seen := make(map[string]struct{}, len(protos))
	for _, p := range protos {
		if p == nil || p.Spec.Path == "" {
			continue
		}
		root := p.Spec.Path
		if !p.Spec.IsImportPath {
			root = filepath.Dir(root)
		}
		root = filepath.Clean(root)
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}
