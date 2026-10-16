# One playbook run

This page follows one run from the click in the browser to the final status. File paths are under `app/ytsoar/`.

```mermaid
sequenceDiagram
    autonumber
    actor B as Browser
    participant H as http/handlers/playbook.go<br/>Trigger
    participant O as application/playbooks/orchestrator.go<br/>runPlaybook
    participant Q as adapters/jobs/jobs.go<br/>Enqueuer
    participant DB as Postgres<br/>(history tables + river_job)
    participant RW as adapters/jobs/jobs.go<br/>RunWorker.Work
    participant L as orchestrator.go<br/>RunLoader
    participant X as application/execution/executor.go<br/>Executor
    participant SB as Sandbox<br/>(grpcruntime/sandbox_runtime.go<br/>SandboxRuntime)
    participant N as adapters/pgstatus/status.go<br/>Notifier + Listener
    participant WS as adapters/ws/hub.go<br/>Hub

    B->>H: POST /playbooks/v1/trigger/{id}
    H->>O: TriggerPlaybook
    rect rgb(235, 245, 255)
    Note over O,DB: One transaction (repository/tx_manager.go)
    O->>DB: CreatePlaybookHistory (status = queued)
    O->>DB: CreateTaskHistory (status = pending)
    O->>Q: EnqueueRun(run ID)
    Q->>DB: INSERT river_job (kind playbook_run)
    end
    O-->>B: 202 Accepted + WebSocket event "queued"

    RW->>DB: River takes the job
    RW->>L: LoadRunMessage(run ID)
    L->>DB: Read history rows, edges and input
    L-->>RW: TaskMessage (graph, tasks, edges, input)
    RW->>X: Run(message)
    X->>DB: queued -> in_progress (guarded UPDATE)
    X->>N: Publish playbook_status (row ID)
    N-->>WS: LISTEN gets the ID, loads the row
    WS-->>B: WebSocket: in_progress

    loop Each node (processNode)
        X->>DB: Task pending -> in_progress
        alt Built-in connector (http_request, condition)
            X->>X: Run in the worker (goconnectors/builtin_runtime.go)
        else Other connector or code snippet
            X->>SB: gRPC ExecuteOperation
            SB-->>X: Result
        end
        X->>DB: Task -> success / failed / skipped
        X->>N: Publish task_status (row ID)
        N-->>WS: LISTEN gets the ID, loads the row
        WS-->>B: WebSocket: task status
    end

    X->>DB: in_progress -> success / failed
    X->>N: Publish playbook_status
    N-->>WS: LISTEN gets the ID, loads the row
    WS-->>B: WebSocket: final status
    X-->>RW: Return (error or nil)
    RW->>DB: River marks the job completed (Work returns nil)
```

## Step by step, with files

| Step | What happens | File | Function |
|---|---|---|---|
| 1 | The browser calls the trigger route. | `internal/adapters/http/handlers/playbook_routes.go`, `playbook.go` | `Trigger` |
| 2 | The service reads the playbook, its tasks and its edges. | `internal/application/playbooks/orchestrator.go` | `runPlaybook` |
| 3 | The service saves the run as `queued` and the tasks as `pending`. | `internal/adapters/repository/playbook_repository.go`, `task_repository.go` | `CreatePlaybookHistory`, `CreateTaskHistory` |
| 4 | The service queues the job in the same transaction. | `internal/adapters/jobs/jobs.go` | `Enqueuer.EnqueueRun` |
| 5 | The worker process takes the job. | `internal/adapters/jobs/jobs.go` | `RunWorker.Work` |
| 6 | The worker builds the full message from the saved rows. | `internal/application/playbooks/orchestrator.go` | `RunLoader.LoadRunMessage` |
| 7 | The executor moves the run to `in_progress`. | `internal/application/execution/executor.go` | `Run`, `setRunStatus` |
| 8 | The executor runs the nodes in order, and runs independent nodes at the same time. | `internal/application/execution/executor.go` | `processNode` |
| 9 | The resolver picks where a node runs: in the worker or in the sandbox. | `internal/application/execution/resolver.go` | `StaticResolver.Resolve` |
| 10 | Each status change goes to Postgres, then out as an event. | `internal/adapters/pgstatus/status.go` | `Notifier.Publish`, `Listener.forward` |
| 11 | The hub sends the event to the browser. | `internal/adapters/ws/hub.go` | `Hub.Broadcast` |

## Things to know

- The job holds only the run ID. The worker builds the rest from the database (step 6).
- Steps 3 and 4 are one transaction. A run never exists without its job.
- Steps 7 and 8 use guarded `UPDATE` statements. See [statuses.md](statuses.md).
- The browser gets the `queued` event from the API directly (step 4). All other events come from `LISTEN/NOTIFY`.
