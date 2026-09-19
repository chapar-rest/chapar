package langsrv

// Pre/post-request scripts run inside the python-executor, which exec()s them
// with `chapar`, `request`, and `response` already defined
// (github.com/chapar-rest/python-executor, main.py). Pyright knows none of
// that, so without these stubs every script is covered in "undefined
// variable" errors. They live in the virtual workspace pyright is started in:
// __builtins__.pyi adds names to every file in it, and chapar.pyi types the
// module (scripts may also `import chapar`).

const chaparStub = `"""Chapar scripting API, available as the global ` + "`chapar`" + `."""
from typing import Any, Callable, Optional

class Request:
    method: str
    url: str
    headers: dict[str, str]
    metadata: dict[str, str]
    params: dict[str, Any]
    query: dict[str, Any]
    trailers: dict[str, str]
    data: Any
    def json(self) -> Any: ...

class Response:
    status_code: int
    headers: dict[str, str]
    text: str
    def json(self) -> Any: ...

def get_env(name: str) -> Any:
    """Return the value of a variable in the active environment."""
    ...

def set_env(name: str, value: Any) -> None:
    """Set a variable in the active environment."""
    ...

on_response: Optional[Callable[[Response], None]]
"""Set to a function to run it with the response once the script finishes."""

print_outputs: list[str]
`

const builtinsStub = `import chapar as chapar
from chapar import Request, Response

request: Request
"""The request being sent."""

response: Response
"""The response (post-request scripts)."""
`

const pyrightConfig = `{
  "typeCheckingMode": "basic",
  "reportMissingImports": "warning",
  "reportMissingModuleSource": "none"
}
`

// workspaceFiles are written into the virtual workspace directory.
var workspaceFiles = map[string]string{
	"chapar.pyi":         chaparStub,
	"__builtins__.pyi":   builtinsStub,
	"pyrightconfig.json": pyrightConfig,
}
