# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Chapar

Chapar is a native cross-platform API testing tool (similar to Postman) built with Go and the [Gio](https://gioui.org) GUI library. It supports HTTP/REST, gRPC, and GraphQL protocols. Data is stored locally on the filesystem — no server component.

## Commands

**Run locally:**
```bash
go run .
make run
```

**Install build dependency (gogio):**
```bash
make install_deps
# or: go install gioui.org/cmd/gogio@latest
```

**Build (requires gogio):**
```bash
make build_macos_app   # macOS app bundles (amd64 + arm64)
make build_linux       # Linux amd64 binary
make build_windows     # Windows binaries (amd64, i386, arm64)
```

**Test and lint (run inside Docker via `chapar/builder:0.1.5` image):**
```bash
make test    # go test -v ./...
make lint    # golangci-lint with .golangci-lint.yaml config
```

To run tests directly without Docker (if CGO dependencies are available):
```bash
go test -v ./...
go test -v ./internal/...   # single package subtree
go test -v -run TestName ./path/to/package  # single test
```

## Architecture

### Layer overview

```
main.go                  → window init, bootstraps ui/app
ui/                      → all Gio UI code
internal/                → business logic, no UI dependencies
vendor/                  → vendored Go dependencies
```

### `internal/` packages

| Package | Responsibility |
|---|---|
| `domain/` | Core data models: workspaces, collections, requests (HTTP + gRPC), environments, proto files |
| `state/` | In-memory state management; UI reads/writes go through here |
| `egress/` | Sends actual network requests — `rest/`, `grpc/`, `graphql/` subdirectories |
| `repository/` | Filesystem persistence (JSON files, v2 format) |
| `importer/` | Import from OpenAPI/Swagger, Postman collections, `.proto` files |
| `scripting/` | Pre/post-request script execution — Python via Docker container, JS via `gval` |
| `variables/` | Environment variable substitution in requests |
| `jsonpath/` | JSONPath evaluation for extracting values from responses |
| `codegen/` | Code generation from request definitions |
| `prefs/` | User preferences persistence |

### `ui/` packages

The UI is built entirely with **Gio** (immediate-mode GPU-rendered GUI). There is no web renderer or Electron.

| Package | Responsibility |
|---|---|
| `ui/app/` | Top-level application orchestration, wires pages + state |
| `ui/pages/` | Full-page views: `requests/`, `environments/`, `protofiles/`, `workspaces/`, `settings/` |
| `ui/widgets/` | Reusable Gio widgets: code editor, fuzzy search, key-value editor, dropdowns, etc. |
| `ui/chapartheme/` | Theme colors and styling constants |
| `ui/navigator/` | Page navigation/routing |
| `ui/modals/` | Modal dialogs |
| `ui/notifications/` | Toast notification system |
| `ui/console/` | Log console panel |
| `ui/explorer/` | Collection/file tree explorer in sidebar |

### Key data flow

1. **State** (`internal/state/`) is the source of truth; UI components read from and dispatch changes to state.
2. **Repository** (`internal/repository/`) loads/saves domain objects to disk; state calls repository.
3. **Egress** (`internal/egress/`) is called when the user sends a request; results flow back through state to UI.
4. **Variables** are resolved at send time by substituting `{{variableName}}` placeholders from the active environment.

### Gio rendering model

Gio uses an immediate-mode rendering loop — every frame, the UI is re-laid-out and re-drawn by calling `Layout()` on components. There is no retained widget tree. Widget state (e.g. text field contents, scroll position) must be stored explicitly in structs. New UI components should follow this same pattern.
