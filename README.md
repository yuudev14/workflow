Initial repo resides here: https://github.com/orgs/yuudev14-workflow/repositories

# YTSoar

YTSoar is a self-hosted SOAR platform. You draw a playbook, press run and watch it work.

**Contents:** [What is YTSoar?](#what-is-ytsoar) · [Quickstart](#quickstart) · [How it works](#how-it-works) · [Writing connectors](#writing-connectors) · [Branching](#branching) · [Development](#development) · [Repository layout](#repository-layout) · [Roadmap](#roadmap)

## What is YTSoar?

SOAR means Security Orchestration, Automation and Response. Security teams do the same steps again and again. For example, they check an IP address, block it and send a message. YTSoar does these steps for you.

You build a **playbook** in a visual editor. A playbook is a graph of steps. You trigger the playbook and watch each step run live.

Example playbook:

```
Alert arrives  ->  Check the IP address  ->  Is the IP bad?  --yes-->  Block the IP  ->  Send a message
                                                  |
                                                  +----no---->  Close the alert
```

### Key words

| Word | Meaning |
|---|---|
| Playbook | A graph of steps that you build in the editor. |
| Node | One step in a playbook. |
| Connector | A small program that a node calls. It talks to a tool, or it runs your code. |
| Run | One execution of a playbook. A run has a status: `queued`, `in_progress`, `success` or `failed`. |
| Worker | The program that runs the nodes of a run. |
| Sandbox | A locked box where user code runs. It has no credentials. |

## Quickstart

You need Docker and Node.js.

```bash
# backend stack (postgres, keycloak, nginx, and the api, worker and sandbox in one process)
make rebuild-containers

# frontend (dev server on :3000, proxied by nginx on :9999)
cd app/frontend && npm install && npm run dev
```

Open http://localhost:9999. Create a playbook, add nodes and trigger it. The statuses appear live.

Ports: API `:8080`, Postgres `:5433`, nginx `:9999`.

In dev, one Go program (`cmd/dev`) runs the api, the worker and the sandbox in one process. One `air` builds it again after each change. This mode is for dev only: the sandbox can read the database settings. Production runs the three programs apart.

- `make start-split` runs the three programs in three containers, as in production (`docker/split.docker-compose.yml`).
- `make dev` runs the one process on your computer, without a backend container. See below.

### Run the backend on your computer

Use `make dev` when you want to run the backend without a backend container. Postgres and Keycloak still run in Docker.

1. Install the tools: `air`, `goose`, Python 3 with `jinja2`, `requests`, `colorlog` and `pydantic-settings`, and Node 24 with `nunjucks` on `NODE_PATH`.
2. Add the Go bin folder to your `PATH`. Without it, the shell cannot find `air` and `goose`.

   ```bash
   go install github.com/air-verse/air@latest
   go install github.com/pressly/goose/v3/cmd/goose@latest
   echo 'export PATH="$PATH:$HOME/go/bin"' >> ~/.zshrc   # then open a new terminal
   ```

3. Make the settings file. `make dev` and `air` read it.

   ```bash
   cp app/ytsoar/.env.sample app/ytsoar/.env
   ```

4. Start the backend. This starts Postgres and Keycloak, runs the migrations and starts `air`.

   ```bash
   make dev
   ```

5. Start the frontend in a second terminal. `make dev` does not start it.

   ```bash
   make start-frontend
   ```

Do not run `make dev` together with `make start-dev` or `make rebuild-containers`. The backend would run twice.

Two files build the backend images. `app/ytsoar/Dockerfile.dev` has the dev stages. They are `dev` (one process), `dev-api`, `dev-worker`, `dev-sandbox` and `debug`. `app/ytsoar/Dockerfile` has the production stages. They are `api`, `worker` and `sandbox`. One `build` stage compiles the three binaries for them.

## How it works

The system has four parts. Three parts are Go programs. They come from one module (`github.com/yuudev14/ytsoar`). The fourth part is the React frontend (Vite).

```
┌──────────┐   REST + WS    ┌─────────────┐   sqlc/pgx   ┌──────────────┐
│ Frontend │ ─────────────> │ Playbook API│ ───────────> │  PostgreSQL  │
│ Vite app │ <──── WS ───── │  cmd/api    │ <─ LISTEN ── │ history +    │
└──────────┘    (nginx)     └─────────────┘   /NOTIFY    │ River jobs   │
                                                          └──────┬───────┘
                                                                 │ work job
                                                           ┌─────┴────┐
                                                           │  Worker  │
                                                           │cmd/worker│
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

1. **Playbook API** (`cmd/api`) - The API gives the REST and WebSocket interfaces. It saves playbooks. When a user triggers a playbook, the API queues a run in Postgres (River). The API sends status updates to the browser.
2. **Worker** (`cmd/worker`) - The worker takes queued runs and runs the graph node by node. It runs independent nodes at the same time. For most nodes, the worker sends the work to the sandbox over gRPC and saves the result. Two first-party connectors (`http_request` and `condition`) are trusted. The worker runs them directly.
3. **Sandbox** (`cmd/sandbox`) - The sandbox runs connectors and code snippets. Each run gets a new `python3` or `node` process. The process has no credentials, a clean environment and a time limit. Untrusted code cannot reach the database or the job queue.
4. **Frontend** - The visual editor. The Vite dev server serves it behind nginx.

The diagrams are in [docs/diagram](docs/diagram):

- [One playbook run](docs/diagram/run-flow.md), with the files for each step.
- [Run and task statuses](docs/diagram/statuses.md).
- [What the worker does with a job](docs/diagram/job-results.md).
- [How the worker starts](docs/diagram/worker-startup.md).
- [How the API starts](docs/diagram/api-startup.md).

Other docs:

- [Worker guide](docs/worker-guide.md): everything the worker does, with examples. Start here.
- [River guide](docs/river-guide.md): how jobs work here: `RunArgs`, `RunWorker`, `NewRunWorker` and `NewWorkerClient`.
- [Ports and interfaces](docs/ports-and-interfaces.md): what each interface is, why we made it, and who uses it.
- [The runtime resolver](docs/runtime-resolver.md): how a node picks where it runs.
- [Auth](docs/auth.md): sign-in, sessions, SSO, users, roles, teams and permissions.
- [Database](docs/database.md): the tables as diagrams, and the enums.
- [Connector upload](docs/connector-upload.md): how a zip becomes a connector.
- [Connector dependencies](docs/connector-dependencies.md): how each connector gets its own libraries.
- [The condition task](docs/condition.md): how branching works, the limits of the Go template engine, and the plan to move to Python.

### One run, step by step

1. You trigger a playbook. The API saves the run as `queued` and queues a job. Both happen in one transaction.
2. The worker takes the job and sets the run to `in_progress`.
3. The worker runs the nodes. A node with user code goes to the sandbox.
4. The worker saves the status of each node and of the run. It sends an event with Postgres `LISTEN/NOTIFY`.
5. The API gets the event and sends it to your browser over WebSocket.

### Why three programs?

The split is about trust, not language. Code that uses credentials runs in Go, outside the sandbox. Code that users supply runs inside the sandbox.

## Writing connectors

Connectors are in `app/connectors/<id>/`. A new connector works on the next run. You do not need to restart.

```
app/connectors/my_connector/
├── info.json          # editor data; "runtime": "python" | "node"
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
├── connector.ts       # … or typescript (Node removes the types, so no build step
│   # is needed. Use the .ts extension in the require call)
│   # const { Connector } = require("../core/connector.ts");
│   # class MyConnector extends Connector {
│   #   async execute(configs: Configs, params: Params, operation: string) { ... }
│   # }
│   # module.exports = { MyConnector };
└── configs/prod.toml  # named TOML configs, one for each task
```

Each language uses the same shape. Write a class that extends the base class in `core/`. The class has one method: `execute(configs, params, operation)`.

- `configs` - The parsed TOML config.
- `params` - The parameters of the task. The system already applied the templates.
- `operation` - The operation name from `info.json`.

If a connector has both `connector.ts` and `connector.js`, the system uses the `.ts` file. TypeScript must use only erasable syntax. Do not use enums, namespaces or parameter properties. To check the syntax, run `npx tsc --noEmit` in `app/connectors`.

**To start a new connector**, copy a template: `app/connectors/sample` (Python), `sample_js` (JavaScript) or `sample_ts` (TypeScript). You can also upload a zip file with `POST /api/v1/connectors` (multipart field `file`). The API validates the zip file, installs its dependencies and records an audit row. `DELETE /api/v1/connectors/<id>` removes the connector.

A parameter can use the output of an earlier node: `{{ var.steps["node name"] }}`.

For quick logic, use the two built-in code nodes. They do not need a full connector. Use `code_snippet_py` (set `result = ...`) or `code_snippet_js` (`const result = ...`, and `await` works). These nodes need no files. The sandbox implements them directly. Snippets accept plain Python and JavaScript only. They do not accept TypeScript.

### Connector dependencies

A connector can declare its own libraries:

- Python: `<id>/requirements.txt`. The system installs the libraries into `<id>/deps/`.
- JS/TS: `<id>/package.json`. The system installs the libraries into `<id>/node_modules/`.

Run `make connector-deps` to install the libraries. The stack must already run. The command runs pip and npm inside the api container. In this way, the compiled packages match the sandbox image. The system keeps the libraries of each connector separate. Two connectors can use different versions of the same library. Code snippets cannot declare dependencies. They get only the base packages of the image. To add more packages, change the sandbox stages in `app/ytsoar/Dockerfile` and `app/ytsoar/Dockerfile.dev`.

## Branching

The `condition` built-in works like a switch statement. It checks an ordered list of cases. It takes the first case that matches. If no case matches, it takes **else**. It has two modes:

- **Switch** - Each case is a simple comparison: left value, operator (`==`, `!=`, `>`, `contains` and others) and right value. You do not need templates.
- **Switch (advanced expression)** - Each case is a full template expression, for example `{{ var.steps["scan"].score > 90 }}`. The system takes the case when the expression is true.

In the editor, connect the condition node to its destination nodes. Then choose the destination node for each case and for **else**. At run time, only the matching branch runs. The system marks the nodes on the other branches as `skipped`. The skip continues to all nodes below them. A node with several input branches still runs when at least one branch is taken.

## Development

Rules for the code (layers, jobs, database, tests) are in [CLAUDE.md](CLAUDE.md). Read it before you change the backend.

```bash
cd app/ytsoar
go build ./... && go vet ./...     # three programs: api, worker, sandbox
go test ./... -count=1             # the localexec tests run real python3 and node
go generate ./...                  # make the mocks again

make migrate                       # run the goose migrations
make sqlc                          # run after you change db/queries/*.sql
```

Conventions: The code uses sqlc and pgx. Migrations are goose files with a date in the name. Squirrel builds the dynamic list queries. `types.Nullable[T]` supports partial updates. Every port has a mockgen mock. Each part gets the logger from its caller.

### Tests

CI (`.github/workflows/ci.yml`) runs all of these on each pull request. To run them on your computer:

**1. Start Postgres and Keycloak, and make the test databases (one time).** The dev stack gives Postgres on port 5433 (password `password`) and Keycloak on port 8180.

```bash
docker compose -f docker/dev.docker-compose.yml up -d ytsoar_postgres ytsoar_keycloak
docker exec ytsoar_postgres createdb -U postgres ytsoar_it    # Go tests
docker exec ytsoar_postgres createdb -U postgres ytsoar_e2e   # browser tests
```

**2. Install the sandbox packages (one time).** The integration and browser tests run playbooks in a sandbox on your computer. The sandbox runs `python3 -I`, which ignores the user site, so do not use `pip install --user`. Node must be version 24.

```bash
python3 -m pip install --break-system-packages jinja2 requests colorlog pydantic-settings
npm install --prefix ~/.ytsoar-node nunjucks@3
export NODE_PATH=~/.ytsoar-node/node_modules
```

**3. Backend tests and lint.**

```bash
cd app/ytsoar
go test ./...                      # unit tests; the database tests skip

export TEST_DATABASE_URL="postgres://postgres:password@localhost:5433/ytsoar_it?sslmode=disable"
go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir db/migrations postgres "$TEST_DATABASE_URL" up
go test -p 1 ./...                 # all tests; -p 1 because two packages share the job queue
go test ./tests/integration -run 'TestAlerts/POST_/alerts' -v   # one test

go vet ./... && golangci-lint run  # golangci-lint 2.x built with Go 1.26
```

**4. Frontend lint.**

```bash
cd app/frontend
npm ci && npm run lint && npx tsc --noEmit
```

**5. Browser tests (Playwright).** Playwright migrates `ytsoar_e2e`, then starts the api (:8090), the sandbox (:50062), the worker and the frontend (:3000). Stop your dev frontend first: it also uses port 3000, and Playwright uses a server that already runs on the port.

```bash
cd app/frontend
npx playwright install chromium    # one time
DB_PORT=5433 DB_PASSWORD=password npx playwright test
DB_PORT=5433 DB_PASSWORD=password npx playwright test e2e/alerts.spec.ts -g "shows no rows" --headed   # one test
```

### Debugging in Docker

Start the stack with the debug override. It builds the one-process program without optimizations. It runs the program under a headless [Delve](https://github.com/go-delve/delve):

```bash
docker compose -f docker/dev.docker-compose.yml -f docker/debug.docker-compose.yml up -d
```

Then press F5 in VS Code. Choose `Attach to ytsoar_api (docker)` in `.vscode/launch.json` (:2345). Set breakpoints in the api, the worker or the sandbox code. Air still reloads the program when you save. Each reload restarts Delve. Attach again after a rebuild.

## Repository layout

```
app/ytsoar/        Go backend - cmd/{api,worker,sandbox,dev}, internal/, db/
app/connectors/    connector tree (python + js/ts + code-node metadata)
app/frontend/      Vite + React visual editor
app/nginx/         reverse proxy (:9999)
app/proto/         connector_runtime.proto (worker <-> sandbox contract)
docker/            dev compose stack (+ split and debug overrides)
```

## Roadmap

- More built-in Go connectors (transforms and enrichment)
- Alerts and incidents together with playbooks
