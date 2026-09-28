# Test cases: schema and runner design

Status: draft. Replaces the approach in PR #164 (old Gio UI, `internal/state`).

A test case is a YAML file that sends existing requests in order, checks
their responses, and passes values from one step to the next. The same
runner serves the app and a headless `chapar test` command for CI.

## Goals

- Reuse the send pipeline (`sender.Service.Send`): auth, collection headers,
  pre/post scripts, cookies, tunnels, timeline and every protocol work
  without test-specific code.
- A run never changes the user's environment or cookie jar unless the test
  opts in.
- Requests are referenced by ID, so renames don't break tests.
- Assertion values keep their JSON types: `false` is a bool, `201` a number.
- One canonical YAML form, so a structured editor round-trips without loss.

## Schema

Files live in `<workspace>/testcases/<name>.yaml`.

```yaml
apiVersion: v1
kind: TestCase
metadata:
  id: 7f3c1e0a-...            # uuid, assigned on create or import
  name: Create and fetch todo
spec:
  description: Creates a todo, reads it back, cleans up
  tags: [smoke, todos]
  options:
    continueOnFailure: false  # keep running steps after one fails
    timeout: 30s              # default for each step
    persistEnv: false         # write run env changes back to the real env
  variables:                  # run scoped, override env values
    - key: title
      value: buy milk
    - key: token
      from: { osEnv: API_TOKEN }
  setup: []                   # same shape as steps; a failure ends the run
  steps:
    - id: create              # unique within the case; results map to it
      name: Create todo
      request:
        id: 3b9e...           # primary lookup
        ref: Todos/Create     # "Collection/Request"; fallback and display
      with:                   # overrides for this step only
        variables: { title: "{{title}} {{randomInt}}" }
        headers: [{ key: X-Trace, value: test }]
        query: []
        body: ""              # HTTP body / GraphQL query / gRPC message
      timeout: 10s
      retry: { count: 5, delay: 1s }   # re-send until assertions pass
      continueOnFailure: false
      assert:
        - { target: status, op: in, value: [200, 201] }
        - { target: header, key: content-type, op: contains, value: json }
        - { target: body, path: $.data.id, op: exists }
        - { target: body, path: $.data.done, op: eq, value: false }
        - { target: body, path: $.data.tags, op: length, value: 2 }
        - { target: time, op: lt, value: 500 }
      capture:
        - { var: todoId, from: body, path: $.data.id }
        - { var: sid, from: cookie, key: session }
  teardown: []                # always runs, even after a failure or cancel
```

### Assertions

| target     | selector | actual value                                      |
|------------|----------|---------------------------------------------------|
| `status`   |          | HTTP status, or gRPC code for gRPC requests       |
| `header`   | `key`    | response header, case-insensitive                 |
| `cookie`   | `key`    | response cookie value                             |
| `metadata` | `key`    | gRPC response metadata, case-insensitive          |
| `trailer`  | `key`    | gRPC trailer, case-insensitive                    |
| `body`     | `path`   | JSONPath over the decoded JSON body               |
| `text`     |          | raw body text                                     |
| `time`     |          | response time in milliseconds                     |
| `size`     |          | response size in bytes                            |

| op                          | passes when                                          |
|-----------------------------|------------------------------------------------------|
| `exists` / `notExists`      | the key or path is (not) there; JSON `null` exists   |
| `eq` / `ne`                 | deep equality after JSON normalization               |
| `gt` `gte` `lt` `lte`       | numeric comparison; numeric strings count as numbers |
| `in`                        | actual equals one element of the `value` list        |
| `contains` / `notContains`  | substring of a string, or element of an array        |
| `matches`                   | Go regexp matches the text of the actual value       |
| `type`                      | `string number boolean null array object`            |
| `length`                    | length of a string, array or object equals `value`   |

Rules:

- Every op except `exists`/`notExists` fails with "not found" when the
  selector matches nothing.
- `eq` compares a number with a numeric string as numbers, since header
  values are always strings.
- Body numbers decode as float64. Integers above 2^53 lose precision;
  compare those with `text` + `matches` for now.
- Wildcard and deep paths (`$..id`, `$.items[*]`) return a list, which may
  be empty. Use `length` rather than `exists` on them.
- String values, and strings inside an `in` list, expand `{{variables}}`:
  `{ target: body, path: $.data.id, op: eq, value: "{{todoId}}" }`.
- `chapar.test()` results from the request's own scripts are reported as
  assertions with source `script`.
- A step with no assertions passes on any response; only a failed send
  makes it an error.

### Captures

`from` takes the same targets as assertions (`status header cookie metadata
trailer body text`). A captured value is stored as text; non-string values
are stored as JSON, the same way `scripting.EnvText` stores script values.

## Variable resolution

Highest wins:

1. `step.with.variables` (this step only, never leaks)
2. run variables: `spec.variables`, overwritten as the run goes by
   captures and by env changes the requests' own scripts and extract rules
   make
3. run env: a copy of the selected env made at run start, keeping its ID
4. built-ins

Each send gets a copy of the run env with 1 and 2 layered on as env
values, so substitution goes through the existing path for HTTP, GraphQL
and gRPC. Variable values are expanded against the layers below them
first (`title: "milk for {{env}}"`), because the send substitutes env
values only one level deep; unknown names such as built-ins are left for
the send.

After each send, the env changes the request's pre/post script or
set-env/extract rules made are merged into the run variables, so chains
that rely on post-request extraction behave as they do in the app. Those
changes, and only those, are what `persistEnv` writes back: captures and
case variables stay in the run.

## Runner

Package `internal/testrun`, no UI imports.

```go
type Sender interface {
    Send(req *domain.Request, env *domain.Environment) (*egress.Response, error)
}

type Config struct {
    NewSender func() Sender           // one per run; see testrun.NewSender
    Requests  RequestSource           // RequestByID, AllRequests
    SaveEnv   func(*domain.Environment) error // for persistEnv
    Getenv    func(string) string     // default os.Getenv
}

type Options struct {
    Env     *domain.Environment // secrets resolved; copied, never mutated
    Only    []string            // step ids: run one step, re-run failed
    OnEvent func(Event)         // StepStarted, StepFinished, RunFinished
}

func New(cfg Config) *Runner
func (r *Runner) Run(ctx context.Context, tc *domain.TestCase, o Options) *Run
```

`testrun.NewSender(lookup, colls, scripts)` builds a `sender.Service` per
run with no repository (post-request actions don't save the env) and an
in-memory cookie store, so every run starts with an empty jar and never
touches the user's.

Requests resolve by `id`, then by `ref`, where a ref is
`Collection/Request` or a standalone request's name. More than one match
is an error asking for `request.id`.

Per step:

1. Resolve the request by `id`, then `ref`. Not found: status `error`.
2. Clone it and apply `with` for its protocol.
3. Build `stepEnv`.
4. Send with a per-step timeout. On failure, retry after `delay` until
   attempts run out; cancelling the run stops the wait.
5. Merge the send's env changes; evaluate assertions and script tests:
   `passed`, `failed` or `error`.
6. Apply captures to run variables. A capture that finds nothing fails
   the step.
7. On failure without `continueOnFailure`: mark the rest `skipped`, run
   teardown.

A failing setup step skips the steps. Teardown always runs, with a
context that outlives a cancel.

Send takes no context, so a timeout or cancel stops waiting for the send
but can't abort it; the abandoned send finishes on its own against its
own copies of the request and env. Plumbing a context through egress
would make cancel immediate.

At the end, with `persistEnv`, diff `runEnv` against the original env and
write it once. Otherwise the copy is dropped.

## Results

```go
type Run struct {
    ID, CaseID, CaseName, EnvName string
    Status   Status // passed | failed | error | cancelled
    Started  time.Time
    Duration time.Duration
    Steps    []StepResult
}

type StepResult struct {
    Section      string // setup | steps | teardown
    StepID, Name string
    Status       Status // passed | failed | error | skipped | cancelled
    Attempts     int
    Duration     time.Duration
    Request      RequestSummary   // protocol, method, final URL
    Response     *ResponseSummary // status, size, time, body capped at 64 KB
    Assertions   []AssertionResult
    Captures     []CaptureResult
    Message      string // why it failed, errored or was skipped
}

type AssertionResult struct {
    Source           string // "assert" | "script"
    Target, Key, Path, Op string
    Expected, Actual any    // JSON typed; the UI diffs these
    Passed           bool
    Message          string
}
```

`failed` means an assertion or capture failed. `error` means the step
could not run: request not found, transport error, timeout, or a
post-request action that raised. A run is `error` if any step is, else
`failed` if any step is. Pass rate counts only steps that ran.
No history in v1; later runs go to `<workspace>/testruns/<caseID>/<ts>.json`,
keeping the last 20.

## Validation

`Validate(tc, find) []Problem` checks unknown targets and ops, missing
selectors, bad JSONPath or regexp, values of the wrong shape for an op,
duplicate step ids and missing requests.
The editor shows these inline; the CLI refuses to run a case that has any.

## Changes outside the runner

Done in phases 2 and 3.

- Move `ui/sender` to `internal/sender`; it only imports `internal/*` and
  the CLI should not depend on `ui/`.
- Add `ScriptTests []scripting.TestResult` to `egress.Response`. Send
  collapses failed script tests into a timeline string today.
- Build a separate sender for runs with `repo=nil, onEnv=nil`, so the
  request's post-request actions don't persist the env.
- Give runs their own cookie store (`cookies.NewMemoryStore`), or
  `saveCookies` writes into the user's jar.
- `scriptingOn()` reads `prefs.GetGlobalConfig()`; `SetScriptingEnabled`
  lets the CLI's `--scripts` override it.
- Add `Load/Create/Update/DeleteTestCase` to `RepositoryV2` and
  `FilesystemV2`. An import without an id, or with one that already
  exists, gets a new id.

## Bundles

Export (in the editor's title row, or the sidebar menu) writes a
`kind: TestBundle` file: the test case, the requests its steps send and
the ones their pre-request actions trigger, the collections those belong
to (headers and auth only, with just those requests), and optionally an
environment. Secret values are left out unless the user opts in; the keys
left out are listed in `secretsLeftOut`, and the CLI warns about any not
given at run time. A file with secrets is written 0600.

Proto files are not bundled: a gRPC step needs server reflection.

## CLI

```
chapar test [--workspace W] [--env E] [--tag T] [--bail] [--scripts]
            [--env-file F] [--os-env PREFIX] [--var k=v ...]
            [--report junit=out.xml] [--report json=out.json] [--no-color]
            [case|file|folder ... | bundle.yaml ...]
```

Bundle files run without a workspace and cannot be mixed with workspace
cases. The bundle's environment is used unless --env picks another.

Environment values can be set over the chosen environment, in order:
`--env-file` (a chapar environment file or KEY=VALUE lines), `--os-env
PREFIX` (OS variables with the prefix, prefix removed), then `--var`. With
no environment chosen they form one of their own. persistEnv writes back
only what the requests changed, never these values; it needs a workspace
environment.

Package `internal/testcli`, dispatched from `main.go` before the GUI
starts. Flags may come before or after the arguments.

- `--workspace` is a workspace folder (for a workspace committed to a
  repo), or the name of one in the app's data folder. Default: the app's
  active workspace.
- Arguments are case names or IDs in the workspace, test case files, or
  folders of them. None runs every case in the workspace, sorted by name.
  `--tag` keeps cases with any of the given tags.
- Every selected case is validated before anything is sent; any problem
  stops the command.
- `--env` picks an environment by name or ID; default none. Secret values
  are decrypted only when the env has some, since reading the macOS
  Keychain can prompt. A passphrase key is unlocked from
  `CHAPAR_SECRETS_PASSPHRASE`. Values that stay locked are left out with a
  warning, and put back untouched if `persistEnv` saves the env.
- Request scripts run when scripting is on in the app's settings, or with
  `--scripts`. The executor starts on the first script, so runs without
  scripts never start it. A Docker container it starts is left running, as
  the app leaves it.
- Console output streams one line per step, with failed assertions under
  it; colors only on a terminal without `NO_COLOR`.

Exit codes: 0 every case passed; 1 a case failed or errored; 2 bad flags,
workspace, environment or test case files, or a report could not be
written; 130 interrupted (teardown still runs and reports are written).

Test case files in the format of PR #164 fail to load and are skipped with
a YAML warning.

## UI

A Tests page (nav item next to Envs) lists the workspace's test cases;
they open in the shared tab strip like requests. The editor has four tabs:

- Steps: Setup / Steps / Teardown, each a list of step cards: name,
  request picker, run-this-step, move, duplicate, delete. Expanded, a card
  edits assertions (target, key or path, op, value), captures, ID,
  timeout, retries and request overrides (variables, headers, query,
  body). A step still named `step` takes its request's name as its ID.
- Variables: name, value, or an OS variable to read.
- Settings: description, default timeout, continue on failure, save
  environment changes, tags.

A step card narrower than 700px folds its move, duplicate and delete
buttons into a ⋯ menu, and stacks each assertion and capture on two lines.
The card learns its width after layout, so the switch shows a frame later.
`type is` picks the JSON type from a list.
- YAML: the whole case as text. Leaving the tab parses it; an error keeps
  the tab open and shows it.

Assertion values are typed as text: `true`, `false`, `null` and numbers
keep their type, `[a, b]` is a list, quotes force a string.

Problems from validation, and text that does not parse (a timeout of
"soon"), are listed above the editor; Run refuses while there are any.

Run (⌘Enter) uses the active environment. Results stream into the lower
pane: a summary with Re-run failed, one row per step, and the selected
step in three tabs: Checks (assertions and captures), Body (the
pretty-printed response, in its own editor, not inside a scroll view) and
Headers. Step cards
show the last run's status.

## Phases

1. Domain types, validation, assertion and capture engine, with tests.
2. Runner and the sender changes above.
3. CLI with JUnit output.
4. Yoga UI: structured editor, YAML tab, live results from `OnEvent`.
5. Data-driven rows (`spec.data`), run history, suites and tags in the
   sidebar.

## Decisions

- `persistEnv` defaults to false.
- `request.ref` stays as a fallback for hand-written YAML; saving from the
  UI always fills `request.id`.
- No step-level scripts in v1: request scripts plus `chapar.test()` cover it.
