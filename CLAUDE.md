# YTSoar

YTSoar is a SOAR app. The backend is Go (`app/ytsoar`, three binaries: `api`, `worker`, `sandbox`). The frontend is Next.js (`app/frontend`). Postgres is the only data service.

Less LOC is preferred. Be parsimonious.

If you create services, keep an extremely clear service boundary. A service does not need to know the details of the model objects. Ideally the service returns state that the model can then adopt.

In general less files is better than more. Do not create a new file or package when an existing one fits.

Never commit or push unless asked. The user reviews, tests and commits.

## Architecture

Follow the dependency rule. Dependencies point inward only, and each layer sees only types it owns.

- Layers: `domain` (types, rules) <- `application` (services, ports) <- `adapters` (pgx repositories, HTTP, River, WebSocket). `cmd/*/main.go` wires them by hand, no DI container.
- `domain` and `application` never import an adapter or a third-party client library (pgx, gin, River, amqp, an SDK). The only outside packages they use are small ones such as `uuid`.
- A service depends on small interfaces (ports) that use domain types. The adapter implements the port. The adapter, not the service, converts between the library structs and the domain types. A library error or struct never crosses a port.
- An adapter does not import another adapter. If it needs something from one, `main.go` injects it as an interface or a function (for example `jobs.NewEnqueuer(pool, repository.TxFromContext)`).
- Build every dependency in `main.go` and pass it in through the constructor. Do not create a client inside a service, and do not use a package-level global.
- Mocks come from `//go:generate mockgen` lines next to the interface.
- Queries are sqlc (`db/queries/*.sql`). After you change a query or a migration, run `make sqlc` (inside `app/ytsoar`). Never edit generated files in `db/*.sql.go`.
- Migrations are goose files in `app/ytsoar/db/migrations`. `ALTER TYPE ... ADD VALUE` needs `-- +goose NO TRANSACTION`. Keep new enum values in `db/enums.sql` too.

## Jobs (River)

- Jobs are thin wrappers. A worker calls one executor method and handles errors (log, then return). No business logic in a worker.
- A job carries only ids. The worker loads the rest from the database, so `river_job` stays small.
- Queue a job with `contracts.RunEnqueuer` inside the same transaction as the rows it depends on. Never queue after the commit.
- A run that failed is not a job error. The executor saves the failure, so the worker returns nil. Returning the error would make River retry and run the connectors again.
- `internal/adapters/jobs/jobs.go` and `db/migrations/*_river.sql` are River's. Do not write sqlc queries against `river_*` tables. Regenerate the migration with `river migrate-get`, never edit it by hand.

## State machine

- Run statuses: `queued -> in_progress -> success|failed`. Task statuses: `pending -> in_progress -> success|failed`, or `pending -> skipped`.
- The allowed moves are in `domain.RunTransitions` and `domain.TaskTransitions`. The repositories pass them to the `UPDATE ... WHERE status = ANY(...)`, so the database refuses a bad move. A refused move returns `domain.ErrInvalidTransition`.
- To add a status, change the transition table, the enum migration, `db/enums.sql` and the frontend `StatusPill` and filters together.
- Status events go over Postgres `LISTEN/NOTIFY` (`internal/adapters/pgstatus`). The payload is an id only, because a NOTIFY payload is capped at 8000 bytes.

## Tests

- Always run `go build ./... && go vet ./... && go test ./...` in `app/ytsoar` after a change.
- SQL guards and the job queue need a real Postgres. Those tests skip unless `TEST_DATABASE_URL` is set. Mocks hide SQL type errors, so run them when you change a query.
- Test with few cases, in the existing test file, and reuse the existing fixtures.

## Gotchas

- `app/frontend/src/services/playbooks/playbooks.schema.ts` has CRLF line endings. Keep them when you edit the file.
- `executor.Run` dumps the whole task message at debug level on every run. This is costly at high volume.

## Reporting

Report to the user only in ASD-STE100 Simplified Technical English. Use short sentences, active voice, simple tenses, and one instruction per sentence. Use approved words with one meaning each. Do not use jargon, idioms, or noun clusters of more than three words. This applies to chat replies, not to code or commit messages.
