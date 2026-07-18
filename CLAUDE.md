# CLAUDE.md — YTSoar

YTSoar is a SOAR platform. The main concept is the **playbook**: a DAG of tasks
(connector operations and code snippets) built in a visual editor and executed
by the Go worker. "Playbook" replaced the old word "workflow" — never use
"workflow" in names, routes, tables, or wire formats.

## Architecture

The boundary between services is **trust**, not language. Three Go binaries in
ONE module (`github.com/yuudev14/ytsoar`, dir `app/ytsoar`) plus the frontend:

| Service | Binary | Credentials | Role |
|---|---|---|---|
| `ytsoar_api` | `cmd/api` | DB + MQ | REST `/api/playbooks/v1` + `/api/connectors/v1`, WS `/ws/playbook`, publishes triggers, forwards status events to WS |
| `ytsoar_playbook_worker` | `cmd/worker` | DB + MQ | consumes queue `playbook`, runs the DAG (Kahn), saves history, publishes status. **Never runs user code.** |
| `ytsoar_sandbox` | `cmd/sandbox` | **none** | gRPC `ConnectorRuntime` on `:50052`; runs ALL dynamic code (connectors + code snippets) as fresh subprocesses |
| frontend | Next.js | — | visual editor, `app/frontend` |

Flow: UI → API (saves history in one tx, publishes `TaskMessage` to queue
`playbook`) → worker (per node: gRPC `ExecuteOperation` to `SANDBOX_ADDR`, OR
in-process for Go builtins) → worker saves results + publishes to fanout exchange
`playbook.status` → API → WS. Wire format
`TaskMessage{graph map[string][]string, tasks, playbook_history_id}` — the
existing fields must never change or be renamed. New fields may only be **added**
and must be optional (`omitempty`) so old producers/consumers keep working; that
is how `edges` (conditional branching, below) was introduced.

### Go builtin connectors (in-worker, trusted)

Some connectors run **in-process in the worker**, not in the sandbox — they are
first-party trusted Go code, so the trust boundary doesn't apply. The worker's
`RuntimeResolver` sends a task to the Go registry
(`internal/adapters/runtimes/goconnectors`) when its `connector_id` is registered
there, otherwise to the sandbox over gRPC. Builtins today: `http_request` and
`condition`. Each has an `info.json` in the connectors tree with `"runtime":
"go"` (metadata only — the registry IS the implementation; `runtime: "go"` is NOT
uploadable, uploads allow python/node only). Params are templated with the
`TemplateEngine` port; the adapter is **gonja** (jinja2 fidelity, matches the
python/node runtimes — do NOT swap to pongo2, it can't parse
`{{ var.steps["node name"].field }}`). TOML configs load exactly like the node
runner.

### Conditional branching

The `condition` builtin is a **switch**: an ordered `cases` list, each with a
stable `id` naming its branch. `result` is the first matching case's `id`, or
`"else"`. Stable ids (not positions) mean deleting a case never misroutes an edge.
Two operations differ only in how a case is written:
- `switch` — simple `{id, left, operator, right}` compare (`==`, `!=`, `>`,
  `contains`, …). A compare error counts as no match, so one bad case can't fail
  the run.
- `switch_expression` — advanced `{id, expression}`, a full template expression
  read for truthiness (already rendered by the registry, so
  `{{ ...score > 80 }}` arrives as `"True"`).

Edges from a condition node carry a `source_handle` naming the branch; the
frontend save payload threads it through, and `PreparePlaybookMessage` puts it on
the additive `TaskMessage.edges []EdgeRef{source, destination, source_handle}`
field. The executor's `conditionResult` normalizes the source node's output to a
selector string (string as-is; a bool would map to `"true"`/`"false"`) and follows
an edge only when its `source_handle` matches. Directional editor handles
(`source-*`/`target-*`) and nil handles always follow, so ordinary nodes are never
gated. A node whose dependencies all completed without a single followed incoming
edge is **skipped** (new `task_status` enum value `skipped`), and the skip
propagates down its subtree. A join runs if at least one incoming edge was
followed.

The editor (`ConditionNode` + `ConnectorOperation`'s `BranchRouter`) exposes a
single output handle (`CONDITION_OUTPUT_HANDLE = "output"`); the user draws edges
to destination nodes, then picks each branch's destination from a dropdown, which
stamps the branch handle onto that edge. Unrouted/released edges keep the `output`
handle — non-directional and never a branch id, so the executor skips them rather
than always running them.

### How the sandbox runs code (localexec)

Every run is a fresh subprocess: a harness script embedded with `go:embed` is
passed inline (`python3 -I -c` / `node --input-type=commonjs-typescript -e`).
Payload goes in as JSON on **stdin**, result comes back as JSON on **stdout**.
Each run gets its own process group (killed on timeout) and a scrubbed env —
user code must never see `DB_*`/`MQ_*`.

**stdout is the result channel.** The harnesses send prints/console output to
stderr. Never add stdout writes to the harnesses or to
`app/connectors/core/connector.py`.

The node harnesses are TypeScript, run via Node's **native type stripping**
(needs Node ≥ 23.6 — the image installs `nodejs-current`). No build step;
only erasable TS syntax (no enums/namespaces/parameter properties —
`erasableSyntaxOnly` in the tsconfigs enforces this in the editor).

Sandbox routing (`cmd/sandbox`):
- `code_snippet_py` → python code harness, `code_snippet_js` → node code harness.
  These are "virtual connectors": only info.json exists in the tree; the
  harness IS the implementation. Snippet code itself is python/JS only — TS in
  a snippet won't strip (it runs through `new AsyncFunction`).
- info.json `"runtime": "node"` + `connector.ts` or `connector.js` → node
  connector harness (TS first).
- everything else → python connector harness.

Templates in params use `{{ var.steps["<node name>"] }}` (jinja2 for Python,
nunjucks for JS/TS).

### Connectors tree (`app/connectors/`)

Layout: `<id>/{info.json, connector.py|connector.ts|connector.js, configs/*.toml}`.
The API is the only writer: `POST /api/connectors/v1` uploads a zip
(validated: info.json + id pattern + runtime entry file + traversal/size
checks; reserved ids core/code_snippet*), extracts it into the tree, runs the
dep installs, and upserts an audit row in the `connectors` table (checksum,
uploaded_by, enabled). `DELETE /api/connectors/v1/:id` removes both. The
sandbox mounts the tree read-only.

**All three languages use the same class contract** (user wants symmetry):
- Python: class inheriting `connectors.core.connector.Connector`
- JS/TS: exported class extending `Connector`. The implementation lives in
  `core/connector.ts`; `core/connector.js` is a 1-line shim so plain-JS
  `require("../core/connector")` keeps working (extensionless require never
  resolves `.ts`). TS connectors must `require("../core/connector.ts")`
  explicitly. The core also owns entry resolution (connector.ts, then
  connector.js), class discovery (`getClassContainer`) and templating
  (`evaluateParams`); the harness just calls it.
- Templates to copy: `app/connectors/sample` (py), `sample_js` (js), `sample_ts` (ts).
- New connectors work on the next run — no restart needed.
- Per-connector deps: `<id>/requirements.txt` → `<id>/deps/` (harness prepends
  it to sys.path), `<id>/package.json` → `<id>/node_modules/` (Node resolves
  it natively). Installed by `make connector-deps` INSIDE the api container
  (musl wheels must match the sandbox image — never pip-install these on the
  host). Vendoring means connectors can pin conflicting versions safely.
  Code snippets have no deps mechanism — image baseline only.
- The `package.json`/`tsconfig.json` in `app/connectors/` and in the harness
  dir are **editor-only** (@types/node etc.) — nothing there is needed at
  runtime; `node_modules` are gitignored.

## Code architecture (hexagonal, ai_gateway style)

`internal/domain` → `internal/application` (services; **ports live in the
consumer's package** as `ports.go`; shared ports in `application/contracts`)
→ `internal/adapters`. Composition roots only in `cmd/*/main.go`. Manual
constructor injection. `internal/logger.Logger` is injected everywhere — no
globals; tests use `logger.NewNop()`. Every port has `//go:generate mockgen`
(go.uber.org/mock), mocks in `mocks/` subpackages.

DB: sqlc + pgx (`db/sqlc.yaml`, `db/queries/*.sql`), goose migrations with
**date-based filenames**, enums duplicated in `db/enums.sql`. Dynamic
list/filter queries use squirrel + the Collect helpers in
`internal/adapters/repository/utils.go`. Partial updates use
`types.Nullable[T]{Value *T, Set bool}` with
`CASE WHEN sqlc.arg('x_set') THEN sqlc.narg('x') ELSE x END`. Transactions:
`contracts.TxManager.WithinTransaction` (tx stored in context; repos pick it
up via `queriesFromContext`).

## Gotchas (do not rediscover these)

- Trigger types are a **static enum** (`manual`, `webhook`, `referenced`,
  `on_create`, `on_update`, `on_delete`) — there is NO `playbook_triggers`
  table and no `/triggers` endpoint. The list exists in three places that must
  stay in sync: the `trigger_type` pg enum (init migration + `db/enums.sql`),
  `domain.TriggerType` constants, and the frontend's `settings/triggers.ts`.
  `playbooks.trigger_parameters` (jsonb) carries the `on_*` options as an
  opaque blob (e.g. `{module: "alert"}`).
- gin query params: uuid fields must bind as `*string` +
  `binding:"omitempty,uuid"`. `uuid.UUID` is `[16]byte` and gin's binder
  fails on it.
- `domain.IsAcyclicGraph` returns **true when there IS a cycle** and
  **mutates its input** — always pass a copy.
- Connectors API returns BARE JSON (no wrapper) — the frontend depends on the
  exact info.json shape. `domain.ConnectorInfo` stays a map so unknown fields
  pass through.
- Tests live in external `_test` packages. Run `go test ./... -count=1` from
  `app/ytsoar` and show fresh output at checkpoints. localexec tests run real
  `python3`/`node`; templating tests skip when jinja2/nunjucks are missing.
- The codebase uses `any`, not `interface{}` — keep it consistent.
- After big renames, trust CLI `go build` over stale gopls/VS Code errors;
  suggest restarting the Go language server.
- Sandbox image deps for python connectors: jinja2, requests, colorlog,
  pydantic-settings (see `app/ytsoar/Dockerfile.dev`). More packages = extend
  that Dockerfile. Node finds nunjucks via
  `NODE_PATH=/opt/ytsoar/harness/node_modules`.
- The image must install `nodejs-current`, NOT `nodejs` (alpine's nodejs is
  22.x; TypeScript type stripping needs ≥ 23.6). Tests that execute the node
  harnesses gate on `requireTypeScriptNode`.
- Node's extensionless `require("../core/connector")` can never resolve a
  `.ts` file — that's why `core/connector.js` exists as a shim. Don't delete
  it, and don't add a second export path (require-cache identity is what
  makes `instanceof Connector` work across JS and TS connectors).
- CLI tools live in `$(go env GOPATH)/bin`: protoc, sqlc, goose, mockgen.
- Go commands run from `app/ytsoar` (the module root is not the repo root).
- Debugging in docker: `docker compose -f docker/dev.docker-compose.yml -f
  docker/debug.docker-compose.yml up` runs each Go service under headless
  dlv (api :2345, worker :2346, sandbox :2347) — attach via
  `.vscode/launch.json`. Air reloads restart dlv → re-attach. The api debug
  entrypoint must keep `make migrate` (normal entrypoint.sh does it).
