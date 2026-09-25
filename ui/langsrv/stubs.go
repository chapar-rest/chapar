package langsrv

// Pre/post-request scripts run inside the python-executor, which runs them
// with `chapar`, `request`, and `response` already defined
// (github.com/chapar-rest/python-executor, chapar_api.py). Pyright knows none
// of that, so without these stubs every script is covered in "undefined
// variable" errors. They live in the virtual workspace pyright is started in:
// __builtins__.pyi adds names to every file in it, and chapar.pyi types the
// module (scripts may also `import chapar`). Keep them in step with
// chapar_api.py.

const chaparStub = `"""Chapar scripting API, available as the global ` + "`chapar`" + `."""
from typing import Any, Callable, Iterator, Literal, Optional, TypeVar, overload

_F = TypeVar("_F", bound=Callable[[], Any])

class Pairs:
    """Ordered key/value pairs that read like a dict. Header and metadata keys
    match case-insensitively. A key may repeat: get() returns the first value,
    get_all() every one."""
    def __init__(self, pairs: Any = ..., case_sensitive: bool = ...) -> None: ...
    def get(self, key: str, default: Any = None) -> Any: ...
    def get_all(self, key: str) -> list[str]: ...
    def add(self, key: str, value: Any) -> None:
        """Append another value for key."""
        ...
    def set(self, key: str, value: Any) -> None:
        """Replace every value of key with this one."""
        ...
    def remove(self, key: str) -> None:
        """Remove every value of key; a missing key is fine."""
        ...
    def keys(self) -> list[str]: ...
    def values(self) -> list[str]: ...
    def items(self) -> list[tuple[str, str]]: ...
    def pairs(self) -> list[tuple[str, str]]:
        """Every (key, value), duplicates included, in order."""
        ...
    def to_dict(self) -> dict[str, str]: ...
    def __getitem__(self, key: str) -> str: ...
    def __setitem__(self, key: str, value: Any) -> None: ...
    def __delitem__(self, key: str) -> None: ...
    def __contains__(self, key: object) -> bool: ...
    def __iter__(self) -> Iterator[str]: ...
    def __len__(self) -> int: ...

class GraphQLRequest:
    query: str
    """The GraphQL document."""
    variables: Any
    """The variables: a dict when they are valid JSON, else the raw text."""

class Resolved:
    """The request with {{variables}} filled in, as Chapar sends it. Read-only."""
    url: str
    headers: Pairs
    body: str
    query: Pairs
    graphql: Optional[GraphQLRequest]
    def json(self) -> Any: ...

class Request:
    """The request. A pre-request script may change it; the changes apply to
    this send only, never to the saved request. Fields keep their
    {{variables}}; see resolved for the filled-in values.

    gRPC: url is the server address, method the full method name and
    headers (or metadata) the metadata. GraphQL: see graphql."""
    protocol: Literal["http", "grpc", "graphql"]
    url: str
    method: str
    headers: Pairs
    metadata: Pairs
    """gRPC metadata; the same as headers."""
    body: str
    query: Pairs
    """HTTP query params."""
    path_params: dict[str, str]
    """HTTP path params, the {name} parts of the URL."""
    graphql: Optional[GraphQLRequest]
    resolved: Resolved
    def json(self) -> Any:
        """The body parsed as JSON."""
        ...
    def set_json(self, obj: Any) -> None:
        """Replace the body with obj encoded as JSON."""
        ...

class Cookie:
    name: str
    value: str
    domain: str
    path: str
    expires: str
    secure: bool
    http_only: bool

class Response:
    """The response (post-request scripts). For gRPC, status_code is the gRPC
    code (0 is OK)."""
    protocol: Literal["http", "grpc", "graphql"]
    status_code: int
    status: str
    headers: Pairs
    text: str
    body: str
    elapsed_ms: float
    size: int
    error: Optional[str]
    """Why the request failed, if it did."""
    cookies: list[Cookie]
    metadata: Pairs
    """gRPC response metadata."""
    trailers: Pairs
    """gRPC trailers."""
    ok: bool
    """A 2xx status, or gRPC code 0."""
    data: Any
    """GraphQL: the data field of the body."""
    errors: Any
    """GraphQL: the errors field of the body."""
    def json(self) -> Any: ...
    def cookie(self, name: str, default: Optional[str] = None) -> Optional[str]: ...

class Env:
    """The active environment. Values set that are not text are stored as JSON."""
    name: str
    def get(self, key: str, default: Any = None) -> Any: ...
    def set(self, key: str, value: Any) -> None: ...
    def unset(self, key: str) -> None: ...
    def all(self) -> dict[str, Any]: ...
    def __getitem__(self, key: str) -> Any: ...
    def __setitem__(self, key: str, value: Any) -> None: ...
    def __delitem__(self, key: str) -> None: ...
    def __contains__(self, key: object) -> bool: ...

phase: Literal["pre", "post"]
protocol: Literal["http", "grpc", "graphql"]
request: Request
response: Optional[Response]
env: Env

def get_env(name: str, default: Any = None) -> Any:
    """Return the value of a variable in the active environment."""
    ...

def set_env(name: str, value: Any) -> None:
    """Set a variable in the active environment."""
    ...

def unset_env(name: str) -> None:
    """Remove a variable from the active environment."""
    ...

def log(*args: Any, sep: str = " ") -> None:
    """Print to the request timeline (print() does the same)."""
    ...

def skip(reason: str = "") -> None:
    """Stop the script and do not send the request (pre-request only)."""
    ...

@overload
def test(name: str, fn: Callable[[], Any]) -> bool: ...
@overload
def test(name: str) -> Callable[[_F], _F]: ...
def test(name: str, fn: Optional[Callable[[], Any]] = None) -> Any:
    """Run fn and record whether it passed, or use as a decorator."""
    ...

on_response: Optional[Callable[[Response], None]]
"""Set to a function to run it with the response once the script finishes."""
`

const builtinsStub = `import chapar as chapar
from chapar import Request, Response

request: Request
"""The request being sent."""

response: Response
"""The response (post-request scripts; None before the request is sent)."""
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
