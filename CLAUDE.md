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
`playbook`) → worker (per node: gRPC `ExecuteOperation` to `SANDBOX_ADDR`) →
worker saves results + publishes to fanout exchange `playbook.status` → API →
WS. Wire format `TaskMessage{graph map[string][]string, tasks, playbook_history_id}`
must never change.

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
- `code_snippet` → python code harness, `code_snippet_js` → node code harness.
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
The API is the only writer (upload endpoint is planned — design in
`~/.claude/plans/i-wanted-to-plan-declarative-umbrella.md` §6). The sandbox
mounts it read-only.

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
