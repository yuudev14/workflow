# YTSoar

A self-hosted SOAR (Security Orchestration, Automation and Response) platform.
Build **playbooks** — graphs of connector operations and code snippets — in a
visual editor, trigger them, and watch them run live.

## Architecture

There are four pieces. The first three are Go binaries built from one module
(`github.com/yuudev14/ytsoar`); the fourth is the Next.js frontend.

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

1. **Playbook API** (`cmd/api`) — the REST and WebSocket front door. Saves
   playbooks, queues a run to RabbitMQ when triggered, and streams status
   updates back to the browser.
2. **Worker** (`cmd/worker`) — picks up queued runs and executes the graph,
   node by node, running independent nodes in parallel. For most nodes it
   hands the work to the sandbox over gRPC and saves the result. Two
   first-party connectors (`http_request`, `condition`) are trusted enough to
   run directly in the worker instead.
3. **Sandbox** (`cmd/sandbox`) — actually runs connectors and code snippets.
   Every run gets a brand-new `python3` or `node` process with no credentials,
   a clean environment, and a timeout, so untrusted code can't touch the
   database or queue.
4. **Frontend** — the visual editor, served by Next.js behind nginx.

The split is about **trust, not language**: anything that touches credentials
lives in Go outside the sandbox; anything that runs user-supplied code lives
inside it.

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

Every language follows the same shape: a class that extends the base class in
`core/` and implements one method, `execute(configs, params, operation)`.

- `configs` — the parsed TOML config
- `params` — the task's parameters, already templated
- `operation` — the operation name from `info.json`

If a connector ships both `connector.ts` and `connector.js`, the `.ts` one
wins. TypeScript is limited to erasable syntax (no enums, namespaces, or
parameter properties) — check with `npx tsc --noEmit` in `app/connectors`.

**To start a new connector**, copy a template: `app/connectors/sample`
(Python), `sample_js` (JavaScript), or `sample_ts` (TypeScript). You can also
upload one as a zip via `POST /api/connectors/v1` (multipart field `file`) —
the API validates it, installs its dependencies, and records an audit row.
`DELETE /api/connectors/v1/<id>` removes it.

Params can reference earlier node outputs with `{{ var.steps["node name"] }}`.

For quick one-off logic without writing a full connector, use the two
built-in code nodes: `code_snippet_py` (set `result = ...`) and
`code_snippet_js` (`const result = ...`, `await` works). They need no files
of their own — the sandbox implements them directly. Snippets only accept
plain Python/JavaScript, not TypeScript.

### Connector dependencies

A connector can declare its own libraries:

- Python: `<id>/requirements.txt` → installed into `<id>/deps/`
- JS/TS: `<id>/package.json` → installed into `<id>/node_modules/`

Run `make connector-deps` to install them (the stack must already be
running — this runs pip/npm inside the api container so compiled packages
match the sandbox image). Each connector's dependencies are vendored
separately, so two connectors can pin different versions of the same library
without conflicting. Code snippets can't declare dependencies — they only get
the image's baseline packages (extend `app/ytsoar/Dockerfile.dev` to add
more).

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

The `condition` builtin works like a switch statement: it checks an ordered
list of cases and takes the first one that matches, or falls to **else** if
none do. Two modes:

- **Switch** — a simple comparison per case: left value, operator (`==`,
  `!=`, `>`, `contains`, …), right value. No templating needed.
- **Switch (advanced expression)** — a full template expression per case,
  e.g. `{{ var.steps["scan"].score > 90 }}`, taken when it's truthy.

In the editor, connect the condition node to its destination nodes, then pick
which node each case (and **else**) routes to. At run time, only the matching
branch runs — nodes on branches that weren't taken are marked `skipped`, and
that skip cascades down their subtree. A node fed by multiple branches still
runs as long as at least one of them was taken.

## Roadmap

- More built-in Go connectors (transforms, enrichment)
- Alerts and incidents alongside playbooks
