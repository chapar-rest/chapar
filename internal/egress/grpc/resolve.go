package grpc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/chapar-rest/chapar/internal/domain"
)

// MissingImport is an import statement that none of the configured import
// paths could satisfy. The user has to point us at a directory that holds it
// before the proto files will parse.
type MissingImport struct {
	// Name is the path as written in the import statement, for example
	// "google/api/annotations.proto".
	Name string
}

// MissingImportsError reports proto imports that no configured import path
// satisfies. Callers can unwrap it with errors.As to ask the user where the
// dependencies live instead of showing a raw parse failure.
type MissingImportsError struct {
	Missing []MissingImport
}

func (e *MissingImportsError) Error() string {
	names := make([]string, 0, len(e.Missing))
	for _, m := range e.Missing {
		names = append(names, m.Name)
	}
	if len(names) == 1 {
		return "app: missing proto dependency: " + names[0]
	}
	return "app: missing proto dependencies: " + strings.Join(names, ", ")
}

// ResolveServerProtos parses the proto files a request points at, using the
// request's own import paths. Unresolved imports come back as a
// *MissingImportsError.
func ResolveServerProtos(info domain.ServerInfo) (*protoregistry.Files, error) {
	registry, missing, err := ResolveProtos(info.ProtoFiles, ServerImportPaths(info.ProtoFiles, info.ImportPaths))
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		return nil, &MissingImportsError{Missing: missing}
	}
	return registry, nil
}

// errNotAnImportWeHold tells protoparse that our lookup hook has nothing, so
// it falls through to its built-in copies of the well-known types.
var errNotAnImportWeHold = errors.New("not resolved here")

// ResolveProtos parses files using importPaths as the resolution roots.
//
// Imports that no import path satisfies come back in missing instead of as an
// error, so a caller can ask the user to locate each one and resolve again.
// A non-nil error means something else went wrong, such as a syntax error or
// an unreadable file.
func ResolveProtos(files, importPaths []string) (registry *protoregistry.Files, missing []MissingImport, err error) {
	if len(files) == 0 {
		return nil, nil, errors.New("app: no *.proto files found")
	}

	// protocompile resolves imports on parallel goroutines, so the hook
	// below can run concurrently.
	var mu sync.Mutex
	var unresolved []MissingImport
	seen := map[string]struct{}{}
	parser := protoparse.Parser{
		ImportPaths:      importPaths,
		InferImportPaths: len(importPaths) == 0,
		LookupImport: func(name string) (*desc.FileDescriptor, error) {
			// Only reached when no import path held the file. protoparse
			// serves the well-known types right after this hook, so anything
			// it does not already know is a dependency we must ask about.
			if _, err := protoregistry.GlobalFiles.FindFileByPath(name); err != nil {
				mu.Lock()
				defer mu.Unlock()
				if _, ok := seen[name]; !ok {
					seen[name] = struct{}{}
					unresolved = append(unresolved, MissingImport{Name: name})
				}
			}
			return nil, errNotAnImportWeHold
		},
		// Keep going after the first bad import so one pass collects every
		// missing dependency instead of surfacing them one at a time.
		ErrorReporter: func(protoparse.ErrorWithPos) error { return nil },
	}

	names, err := protoparse.ResolveFilenames(importPaths, files...)
	if err != nil {
		return nil, nil, err
	}

	fds, err := parser.ParseFiles(names...)
	mu.Lock()
	found := append([]MissingImport(nil), unresolved...)
	mu.Unlock()
	if len(found) > 0 {
		// Unresolved imports are the expected failure; report those and let
		// the caller drive the user through fixing them. The compiler stops
		// at the first failed import, so the hook may not have seen the
		// rest; walk the imports to list them all.
		if all, werr := missingImports(names, importPaths); werr == nil && len(all) > 0 {
			return nil, all, nil
		}
		return nil, found, nil
	}
	if err != nil {
		return nil, nil, err
	}

	fdset := &descriptorpb.FileDescriptorSet{}
	walked := make(map[string]struct{})
	for _, fd := range fds {
		fdset.File = append(fdset.File, walkFileDescriptors(walked, fd)...)
	}

	registry, err = protodesc.NewFiles(fdset)
	if err != nil {
		return nil, nil, err
	}
	return registry, nil, nil
}

// missingImports walks the import graph of names, which are relative to
// importPaths, and returns in import order every import that no import path
// holds and that is not a well-known type.
func missingImports(names, importPaths []string) ([]MissingImport, error) {
	var missing []MissingImport
	seen := make(map[string]struct{}, len(names))
	var queue []string
	for _, n := range names {
		seen[n] = struct{}{}
		if path, ok := findInImportPaths(importPaths, n); ok {
			queue = append(queue, path)
		}
	}
	for ; len(queue) > 0; queue = queue[1:] {
		fds, err := protoparse.Parser{}.ParseFilesButDoNotLink(queue[0])
		if err != nil {
			return nil, err
		}
		for _, dep := range fds[0].GetDependency() {
			if _, ok := seen[dep]; ok {
				continue
			}
			seen[dep] = struct{}{}
			if path, ok := findInImportPaths(importPaths, dep); ok {
				queue = append(queue, path)
				continue
			}
			if _, err := protoregistry.GlobalFiles.FindFileByPath(dep); err == nil {
				continue
			}
			missing = append(missing, MissingImport{Name: dep})
		}
	}
	return missing, nil
}

// findInImportPaths returns the file that name resolves to under the first
// import path holding it.
func findInImportPaths(importPaths []string, name string) (string, bool) {
	for _, p := range importPaths {
		path := filepath.Join(p, filepath.FromSlash(name))
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

// ImportRootFor returns the directory to add as an import path so that dir
// satisfies importName. Picking ".../vendor/google/api" for the import
// "google/api/annotations.proto" yields ".../vendor", which is the root protoc
// needs; a directory that does not end in the import's own prefix is returned
// unchanged.
func ImportRootFor(dir, importName string) string {
	dir = filepath.Clean(dir)
	prefix := filepath.Dir(filepath.FromSlash(importName))
	if prefix == "." || prefix == string(filepath.Separator) {
		return dir
	}
	if root := strings.TrimSuffix(dir, string(filepath.Separator)+prefix); root != dir {
		return root
	}
	return dir
}

// ServerImportPaths returns the import roots to parse a request's proto files
// with: the ones the user configured, plus each proto file's own directory so
// a single self-contained file still resolves without any setup.
func ServerImportPaths(protoFiles, importPaths []string) []string {
	out := make([]string, 0, len(importPaths)+len(protoFiles))
	seen := make(map[string]struct{}, len(importPaths)+len(protoFiles))
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range importPaths {
		add(p)
	}
	for _, f := range protoFiles {
		add(filepath.Dir(f))
	}
	return out
}
