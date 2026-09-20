package grpc

import (
	"os"
	"path/filepath"
	"testing"
)

// writeProto writes content to dir/name, creating parent directories.
func writeProto(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const selfContained = `syntax = "proto3";
package demo;
service Greeter {
  rpc Hello (HelloRequest) returns (HelloReply);
}
message HelloRequest { string name = 1; }
message HelloReply { string text = 1; }
`

func TestResolveProtosParsesASelfContainedFile(t *testing.T) {
	dir := t.TempDir()
	path := writeProto(t, dir, "demo.proto", selfContained)

	reg, missing, err := ResolveProtos([]string{path}, ServerImportPaths([]string{path}, nil))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing: %v, want none", missing)
	}
	if reg == nil || reg.NumFiles() == 0 {
		t.Fatal("no files in the registry")
	}
}

func TestResolveProtosAcceptsWellKnownImportsWithoutHelp(t *testing.T) {
	dir := t.TempDir()
	path := writeProto(t, dir, "wkt.proto", `syntax = "proto3";
package demo;
import "google/protobuf/timestamp.proto";
message Event { google.protobuf.Timestamp at = 1; }
`)

	_, missing, err := ResolveProtos([]string{path}, ServerImportPaths([]string{path}, nil))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("well-known types should need no import path, got %v", missing)
	}
}

func TestResolveProtosReportsMissingImports(t *testing.T) {
	dir := t.TempDir()
	path := writeProto(t, dir, "svc.proto", `syntax = "proto3";
package demo;
import "google/api/annotations.proto";
import "common/types.proto";
message Thing { string id = 1; }
`)

	_, missing, err := ResolveProtos([]string{path}, ServerImportPaths([]string{path}, nil))
	if err != nil {
		t.Fatalf("an unresolved import should not be an error: %v", err)
	}
	names := make(map[string]bool, len(missing))
	for _, m := range missing {
		names[m.Name] = true
	}
	// Both are reported in one pass, so the user is not asked one at a time.
	if !names["google/api/annotations.proto"] || !names["common/types.proto"] {
		t.Fatalf("missing = %v, want both unresolved imports", missing)
	}
}

func TestResolveProtosSucceedsOnceTheImportRootIsGiven(t *testing.T) {
	dir := t.TempDir()
	vendor := filepath.Join(dir, "vendor")
	writeProto(t, vendor, "common/types.proto", `syntax = "proto3";
package common;
message ID { string value = 1; }
`)
	path := writeProto(t, dir, "svc.proto", `syntax = "proto3";
package demo;
import "common/types.proto";
service S { rpc Get (common.ID) returns (common.ID); }
`)

	_, missing, err := ResolveProtos([]string{path}, ServerImportPaths([]string{path}, nil))
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("first pass missing = %v, want one", missing)
	}

	// What the dialog does with the folder the user picked.
	root := ImportRootFor(filepath.Join(vendor, "common"), missing[0].Name)
	if root != vendor {
		t.Fatalf("import root = %q, want %q", root, vendor)
	}

	reg, missing, err := ResolveProtos([]string{path}, ServerImportPaths([]string{path}, []string{root}))
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("second pass missing = %v, want none", missing)
	}
	if reg == nil || reg.NumFiles() == 0 {
		t.Fatal("no files in the registry after the import path was added")
	}
}

func TestResolveProtosReportsSyntaxErrorsAsErrors(t *testing.T) {
	dir := t.TempDir()
	path := writeProto(t, dir, "bad.proto", `syntax = "proto3";
message Broken { this is not proto }
`)

	_, missing, err := ResolveProtos([]string{path}, ServerImportPaths([]string{path}, nil))
	if err == nil {
		t.Fatal("a syntax error should be an error, not a missing import")
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v, want none", missing)
	}
}

func TestResolveProtosRejectsAnEmptyFileList(t *testing.T) {
	if _, _, err := ResolveProtos(nil, nil); err == nil {
		t.Fatal("want an error for no proto files")
	}
}

func TestImportRootFor(t *testing.T) {
	sep := string(filepath.Separator)
	cases := []struct {
		name, dir, imp, want string
	}{
		{
			name: "strips the import prefix",
			dir:  sep + filepath.Join("src", "vendor", "google", "api"),
			imp:  "google/api/annotations.proto",
			want: sep + filepath.Join("src", "vendor"),
		},
		{
			name: "keeps a directory that is already the root",
			dir:  sep + filepath.Join("src", "vendor"),
			imp:  "google/api/annotations.proto",
			want: sep + filepath.Join("src", "vendor"),
		},
		{
			name: "import with no directory of its own",
			dir:  sep + filepath.Join("src", "protos"),
			imp:  "types.proto",
			want: sep + filepath.Join("src", "protos"),
		},
		{
			name: "trailing separator does not defeat the match",
			dir:  sep + filepath.Join("src", "vendor", "common") + sep,
			imp:  "common/types.proto",
			want: sep + filepath.Join("src", "vendor"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ImportRootFor(tc.dir, tc.imp); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestServerImportPathsDedupesAndAddsFileDirs(t *testing.T) {
	sep := string(filepath.Separator)
	files := []string{
		sep + filepath.Join("src", "api", "a.proto"),
		sep + filepath.Join("src", "api", "b.proto"),
	}
	vendor := sep + filepath.Join("src", "vendor")

	got := ServerImportPaths(files, []string{vendor, vendor})
	want := []string{vendor, sep + filepath.Join("src", "api")}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
