# YTSoar

YTSoar is a SOAR app. A user builds a **playbook** (a graph of steps) in a visual editor. The app runs the playbook and shows the status live.

The backend is Go (`app/ytsoar`). It has three programs: `api`, `worker` and `sandbox`. The frontend is Next.js (`app/frontend`). Postgres is the only data service.


## Where things are

- `app/ytsoar/cmd/*/main.go`: starts each program and connects the parts.
- `app/ytsoar/internal/app`: connects the parts of each program (`NewAPI`, `NewWorker`, `NewSandbox`). The `cmd/*/main.go` files and the integration tests call it.
- `app/ytsoar/internal/domain`: types and rules.
- `app/ytsoar/internal/application`: services (the business logic).
- `app/ytsoar/internal/adapters`: code that talks to the outside (Postgres, HTTP, River, WebSocket).
- `app/ytsoar/tests/integration`: integration tests of the API, the worker and the sandbox with a real Postgres.
- `app/ytsoar/db`: SQL queries (`queries/`), migrations (`migrations/`) and the generated code.
- `app/connectors`: connectors that playbook steps call.

## Working rules

- Less code is better. Be parsimonious.
- Fewer files is better. Do not add a file or a package when an existing one fits.
- Keep a clear service boundary. A service does not need to know the details of the model objects. Ideally the service returns state that the model can then adopt.
- Never commit or push unless asked. The user reviews, tests and commits.
- Write each PR description with the `/pr-description` skill.
- Do not mix frontend and backend changes in one PR. If an issue needs both, make two branches and two PRs: one for the backend and one for the frontend.
  - Make the backend PR first. The frontend uses the new API.
  - Each PR names the issue and the other PR in its description.
  - The frontend PR closes the issue (`Fixes #<n>`). The backend PR only refers to it (`Refs #<n>`).

## Architecture

The backend has three layers. Each layer knows only the layer inside it.

```
adapters  ->     application     ->    domain
(talk to the     (business             (plain types
 world http      logic, service)       and rules)
 grpc etc)
```

- `domain`: plain Go types and rules, for example the `Playbook` type and the run status rules. It imports nothing from the other layers.
- `application`: services that do the work, for example "run a playbook". A service never talks to the database or to HTTP directly. It calls a **port**. A port is a small Go interface that the service owns.
- `adapters`: code that talks to the outside world, for example Postgres (pgx), HTTP (gin), River and WebSocket. An adapter **implements** a port. It also converts library types to domain types.
- `cmd/*/main.go`: connects everything. It builds each adapter and passes it to the service through the constructor. We use no DI container and no global variable.

Example: queue a playbook run.

1. The service calls `RunEnqueuer.EnqueueRun(...)`. `RunEnqueuer` is a port in `application/contracts`.
2. The adapter `jobs.Enqueuer` implements this port. It calls River.
3. `main.go` gives the adapter to the service. The service never sees River.

Rules:

- `domain` and `application` import no adapter and no third-party client library (pgx, gin, River). `uuid` is fine.
- No library struct or error crosses a port. The adapter converts it first.
- An adapter does not import another adapter. If it needs one, `main.go` passes it in, for example `jobs.NewEnqueuer(pool, repository.TxFromContext)`.
- Put `//go:generate mockgen` next to each port, so tests can use a mock.

## Database

Run all commands in `app/ytsoar`.

**You change a query** (`db/queries/*.sql`):

1. Run `make sqlc`. It makes the Go code again in `db/`.
2. Run `go build ./...`. Fix the callers that the new code breaks.
3. Run the tests with `TEST_DATABASE_URL` set (see Tests).

**You change the schema** (a table, a column or an enum):

1. Make a new goose file in `db/migrations`. Use a date in the name, for example `20260801090000_add_x.sql`. Do not edit an old migration.
2. If you add an enum value, add it to `db/enums.sql` too. sqlc reads the enums from this file.
3. Run `make migrate`. It applies the migration with the `DB_*` settings. Use `make migrate-down` to undo the last one.
4. Run `make sqlc`.
5. Run `go build ./... && go test ./...`.

Rules:

- Never edit `db/*.sql.go`. Run `make sqlc` instead.
- `ALTER TYPE ... ADD VALUE` needs `-- +goose NO TRANSACTION` at the top of the file.

## Mocks

Run all commands in `app/ytsoar`.

**You add or change a port** (a Go interface that tests mock):

1. For a new port, put this line above the interface: `//go:generate mockgen -destination=mocks/<name>_mock.go -package=mocks . <Interface>`
2. Run `go generate ./...`. It makes all the mocks again. The mocks go in a `mocks/` folder next to the port.
3. If the command `mockgen` is missing, run `go install go.uber.org/mock/mockgen@v0.6.0`. Use the version in `go.mod`.
4. Run `go build ./... && go test ./...`. Fix the tests that use the old method.

Never edit a file in `mocks/`. Run `go generate ./...` instead.

## Jobs (River)

A job is a task that a worker does later. River stores the jobs in Postgres.

- Jobs are thin wrappers. All logic belongs in the executor or the service. A worker calls one method and handles errors (log, then return).
- Naming: name the args and the worker after the method they call. For example, `RunArgs` and `RunWorker` call `Executor.Run`, and the job kind is `playbook_run`.
- A job holds only ids. The worker loads the rest from the database.
- Queue a job in the same transaction as the rows it needs, never after the commit.
- A failed run is not a job error. The executor saves the failure, and the worker returns nil. If the worker returns the error, River gives the job to a new attempt.
- A run that cannot start is not retried. The worker calls `Abort` and the run shows `failed` at once. `maxAttempts` is 2. The second attempt is for a job that River takes back (a worker died, the final status was not saved, or `Abort` failed). If the run already started, attempt 2 only marks it `failed`. If it never started, attempt 2 can start it. A run that started is never started again.
- Every goroutine that you start must have a `recover`. River catches a panic only in the job goroutine.
- For a task that must repeat, the service method queues the next job with a delay. Do not loop inside a worker.
- River owns the `river_*` tables and `*_river.sql`. Do not query them or edit them.

## Run and task statuses

| | Allowed moves |
|---|---|
| Run | `queued` -> `in_progress` -> `success` or `failed` |
| Task | `pending` -> `in_progress` -> `success` or `failed`, or `pending` -> `skipped` |

- The rules are in `domain.RunTransitions` and `domain.TaskTransitions`. The database checks every change (`UPDATE ... WHERE status = ANY(...)`). A refused change returns `domain.ErrInvalidTransition`.
- To add a status, change these together: the transition table, the enum migration, `db/enums.sql`, and the frontend `StatusPill` and filters.
- Status events use Postgres `LISTEN/NOTIFY` (`internal/adapters/pgstatus`). The payload is an id only, because a NOTIFY payload is limited to 8000 bytes.

## Tests

- After each change, run this in `app/ytsoar`: `go build ./... && go vet ./... && go test ./...`
- Some tests need a real Postgres (the SQL status rules, the job queue and the status events). They skip unless `TEST_DATABASE_URL` is set. Mocks hide SQL type errors, so run these tests when you change a query.
- Integration tests are in `tests/integration`, one file for each area. `TestMain` starts the api, the worker and the sandbox (`app.NewAPI`, `app.NewWorker`, `app.NewSandbox`) with a real Postgres. The tests send real HTTP requests and sign in as the seeded admin. `runs_test.go` runs playbooks through the worker and the sandbox. Use a new, empty database: the admin seed does nothing when an admin already exists. Run them with `TEST_DATABASE_URL=... go test -p 1 ./...`, because the job queue tests share the queue.
- Name the tests in describe/it style with nested `t.Run`: the outer name is the route or the area (`POST /alerts`), the inner name says what it does (`returns 400 when the title is missing`). Use table cases (`expectAll`) for request and status checks. Put a new test in the file of its area.
- The run tests need Python with `jinja2`, `requests`, `colorlog` and `pydantic-settings` (the sandbox runs `python3 -I`, so not in the user site), and Node with `nunjucks` on `NODE_PATH`.
- Write few test cases. Put them in the existing test file and reuse the existing fixtures.

## Gotchas

- `app/frontend/src/services/playbooks/playbooks.schema.ts` has CRLF line endings. Keep them when you edit the file.
- `executor.Run` writes the whole task message to the debug log on every run. This is costly at high volume.

## Reporting

Report to the user only in ASD-STE100 Simplified Technical English. Use short sentences, active voice, simple tenses, and one instruction per sentence. Use approved words with one meaning each. Do not use jargon, idioms, or noun clusters of more than three words. This applies to chat replies, not to code or commit messages.
