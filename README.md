# YTSoar

A self-hosted SOAR (Security Orchestration, Automation and Response) platform.
Build **playbooks** — graphs of connector operations and code snippets — in a
visual editor, trigger them, and watch them run live.

## Architecture

Services are split by **trust**, not by language. Everything that orchestrates
and holds credentials is Go. All user code and connectors run in a sandbox
that has no credentials, as short-lived isolated subprocesses.

```
┌──────────┐   REST + WS    ┌─────────────┐  trigger   ┌──────────┐
│ Frontend │ ─────────────> │ Playbook API│ ─────────> │ RabbitMQ │
│ Next.js  │ <──── WS ───── │  cmd/api    │ <─ status ─│          │
└──────────┘    (nginx)     └──────┬──────┘   fanout   └────┬─────┘
                                   │ sqlc/pgx               │ consume
                            ┌──────┴──────┐            ┌────┴─────┐
                            │  PostgreSQL │ <──────────│  Worker  │
                            └─────────────┘  history   │cmd/worker│
                                                       └────┬─────┘
                                       gRPC ExecuteOperation│ per node
                                                  ┌─────────┴──────────┐
                                                  │      Sandbox       │
                                                  │    cmd/sandbox     │
                                                  │ ── no credentials ─│
                                                  │ python3 > py conn. │
                                                  │ python3 > py code  │
                                                  │ node    > js conn. │
                                                  │ node    > js code  │
                                                  └────────────────────┘
```

- **Playbook API** (`app/ytsoar/cmd/api`) — REST (`/api/playbooks/v1`,
  `/api/connectors/v1`) and WebSocket (`/ws/playbook`). Publishes triggered
  playbooks to RabbitMQ and pushes status updates to the browser.
- **Worker** (`app/ytsoar/cmd/worker`) — consumes the `playbook` queue and runs
  the graph. Independent nodes run in parallel; any failure stops the run.
  For each node it calls the sandbox over gRPC, saves the result, and
  publishes a status event. It never runs user code itself. First-party **Go
  builtin connectors** (`http_request`, `condition`) are the exception: they are
  trusted Go code and run in-process in the worker, not the sandbox.
- **Sandbox** (`app/ytsoar/cmd/sandbox`) — runs all connectors and code
  snippets. Each run is a fresh `python3` or `node` process with a clean
  environment, its own process group, and a hard timeout. The container has
  no database or queue credentials.

All three binaries live in one Go module (`github.com/yuudev14/ytsoar`) with a
hexagonal layout: `internal/domain` → `internal/application` →
`internal/adapters`, wired together in `cmd/*/main.go`.

## Quickstart

```bash
# backend stack (postgres, rabbitmq, api, worker, sandbox, nginx)
make rebuild-containers

# frontend (dev server on :3000, proxied by nginx on :9999)
cd app/frontend && npm install && npm run dev
```

Open http://localhost:9999 — create a playbook, add nodes, trigger it, and
watch the statuses stream in.

Ports: API `:8080`, RabbitMQ UI `:15673`, Postgres `:5433`, nginx `:9999`.

## Writing connectors

Connectors live in `app/connectors/<id>/`. New connectors work on the next
run — no restart needed.

```
app/connectors/my_connector/
├── info.json          # editor metadata; "runtime": "python" | "node"
├── connector.py       # python …
│   # from connectors.core.connector import Connector
│   # class MyConnector(Connector):
│   #     def execute(self, configs, params, operation, *args, **kwargs): ...
├── connector.js       # … or javascript (same class contract)
│   # const { Connector } = require("../core/connector");
│   # class MyConnector extends Connector {
│   #   async execute(configs, params, operation) { ... }
│   # }
│   # module.exports = { MyConnector };
├── connector.ts       # … or typescript (Node strips the types natively —
│   # no build step; require the core with an explicit .ts extension)
│   # const { Connector } = require("../core/connector.ts");
│   # class MyConnector extends Connector {
│   #   async execute(configs: Configs, params: Params, operation: string) { ... }
│   # }
│   # module.exports = { MyConnector };
└── configs/prod.toml  # named TOML configs, selectable per task
```

All languages follow the same shape: a class extending the base class in
`core/` with an `execute(configs, params, operation)` method. `configs` is the
parsed TOML file, `params` are the task parameters (already templated), and
`operation` is the operation name from info.json. A node connector ships
`connector.ts` **or** `connector.js` (`.ts` wins if both exist); TypeScript
must stick to erasable syntax — no enums, namespaces or parameter properties
(run `npx tsc --noEmit` in `app/connectors` to check).

Copy a template to start: `app/connectors/sample` (Python),
`app/connectors/sample_js` (JavaScript) or `app/connectors/sample_ts`
(TypeScript). Or upload one as a zip (files at the zip root or under one
folder): `POST /api/connectors/v1` with multipart field `file` — the API
validates it, installs its dependencies, and records an audit row;
`DELETE /api/connectors/v1/<id>` removes it.

Params support templating against earlier node outputs:
`{{ var.steps["node name"] }}`.

Two built-in code nodes run raw snippets: `code_snippet_py` (Python, set
`result = ...`) and `code_snippet_js` (JavaScript, `const result = ...`,
`await` works). They need no files besides info.json — the sandbox itself
implements them. Snippets are Python/JavaScript only — TypeScript syntax
won't run there (only connector files get type-stripped).

### Connector dependencies

A connector can declare its own libraries:

- Python: `<id>/requirements.txt` → installed into `<id>/deps/`
- JS/TS: `<id>/package.json` → installed into `<id>/node_modules/`

Install them with `make connector-deps` (the stack must be running — it
executes pip/npm inside the api container so compiled packages match the
sandbox image). Dependencies are vendored **per connector**: two connectors
can pin different versions of the same library without conflict, and a
connector's own version wins over the image's baseline packages. Code
snippets can't declare dependencies — they only see the image's baseline
(extend `app/ytsoar/Dockerfile.dev` for those).

## Development

```bash
cd app/ytsoar
go build ./... && go vet ./...     # three binaries: api, worker, sandbox
go test ./... -count=1             # localexec tests run real python3/node
go generate ./...                  # regenerate mocks

make migrate                       # goose migrations
sqlc generate                      # after editing db/queries/*.sql
```

Conventions: sqlc + pgx, goose date-stamped migrations, squirrel for dynamic
list queries, `types.Nullable[T]` for partial updates, mockgen on every port,
logger injected everywhere.

### Debugging in Docker

Start the stack with the debug override — it rebuilds each Go binary without
optimizations and runs it under a headless [Delve](https://github.com/go-delve/delve):

```bash
docker compose -f docker/dev.docker-compose.yml -f docker/debug.docker-compose.yml up -d
```

Then hit F5 in VS Code with one of the `Attach to …` configs in
`.vscode/launch.json` (api :2345, worker :2346, sandbox :2347) and set
breakpoints. Air still hot-reloads on save; each reload restarts Delve, so
re-attach after a rebuild.

## Repository layout

```
app/ytsoar/        Go backend — cmd/{api,worker,sandbox}, internal/, db/
app/connectors/    connector tree (python + js/ts + code-node metadata)
app/frontend/      Next.js visual editor
app/nginx/         reverse proxy (:9999)
app/proto/         connector_runtime.proto (worker <-> sandbox contract)
docker/            dev compose stack
```

## Branching

The `condition` builtin is a **switch**: an ordered list of cases, and the flow
takes the first one that matches (or the **else** branch when none do). It comes
in two modes:

- **Switch** — each case is a simple comparison: a left value, an operator
  (`==`, `!=`, `>`, `contains`, …), and a right value. No templating to learn.
- **Switch (advanced expression)** — each case is a full template expression,
  e.g. `{{ var.steps["scan"].score > 90 }}`, taken when it reads truthy.

In the editor you connect the condition node to the destination nodes, then in its
settings pick which node each case goes to (plus one for **else**). At run time the
worker follows only the matching branch — nodes on a branch that was not taken
complete as `skipped` and their subtree skips too. A node that several branches
feed into runs as long as at least one of those branches was taken.

## Roadmap

- More built-in Go connectors (transforms, enrichment)
- Alerts and incidents alongside playbooks
