# YTSoar Frontend — User Journey

Snapshot of the current UI so we have a shared reference when asking for
mockups/redesigns. Each screen lists: route, what's on it today, what's
rough/unfinished, and a prompt seed for generating an improved UI image.

## Global shell

Every route renders inside one shell (`src/app/layout.tsx`):
- Left: collapsible icon sidebar (`AppSidebar`) — brand link "YTSoar", one nav
  group "Playbooks" with sub-items **Playbooks**, **Scheduler**, **Connectors**.
  Only **Playbooks** actually routes anywhere (`/playbooks`); Scheduler and
  Connectors are placeholder links (`href="#"`) — no pages exist yet.
- Sidebar footer: dark-mode toggle + a `NavUser` block that is still wired to
  hardcoded sample data (`shadcn` / `m@example.com`, no real auth/profile).
  This is *not* real navigation, and functionally most of the UI depth is a
  single feature (playbook editor) hanging off one sidebar link.

---

## 1. Playbooks list — `/playbooks`

**What's there:** A tab layout (`(view)/layout.tsx`) with an "Overview" header,
a **Create Playbook** button, a row of 5 identical stat tiles hardcoded to
"Success Execution / 5,444" (not wired to real data), and two tabs:
Playbooks / Executions.

The Playbooks tab (`(view)/page.tsx`) lists playbooks as plain bordered rows
(name + static "active" label, no real status), each linking to the editor.
Loading state is 5 skeleton rows.

**Create Playbook:** a dialog (`CreatePlaybookForm.tsx`) with just Name +
Description fields, submits and redirects straight into the editor.

**Rough edges:**
- Stat tiles are fake/duplicated placeholder data.
- "active" status on every playbook card is hardcoded, not derived from state.
- No search, filter, sort, tags, or folders — just a flat list.
- No empty state design (only the loading skeleton exists).

**Image-gen prompt seed:** "Redesign a SaaS automation-platform dashboard list
page: header with title + create button, a real KPI row (e.g. active
playbooks, runs today, success rate, failed runs), a searchable/filterable
table or card grid of playbook rows showing name, trigger type, last run
status badge, last run time. Tabs for Playbooks vs Executions history. Include
an empty state."

---

## 2. Executions — `/playbooks/executions`

**What's there:** Same tab shell as above; a flat list of past playbook runs
(name + `triggered_at` timestamp), linking to a `(broken)` route
`/playbooks/update/[id]` that doesn't match the real editor path pattern
(`/playbooks/[playbookId]`) — likely a stale link.

**Rough edges:**
- No run status/duration/trigger-type column, just name + time.
- No pagination.
- Link path bug (points at a non-existent route).

**Image-gen prompt seed:** "Design an execution-history table: columns for
playbook name, status (success/failed/running badge), trigger source, start
time, duration; row click opens run detail; include filters by status/date."

---

## 3. Playbook editor (canvas) — `/playbooks/[playbookId]`

The core screen. Full-height `@xyflow/react` canvas (`ReactFlowPlayground`)
under a slim top bar (playbook name + description, **Trigger / Delete / Save**
buttons). A floating two-icon rail on the left edge switches between **Editor**
and **Execution (history)** for this playbook.

**Nodes on canvas** (`components/react-flow/nodes/`):
- `StartNode` — a circular "Play" icon avatar, appears once the trigger is
  chosen.
- `PlaybookNode` — connector/task step: avatar with a generic workflow icon,
  connector name (bold) + step name (muted) underneath, hover-reveals a delete
  (×) button, connection handles rendered by `RenderHandles`.
- `ConditionNode` — same card shell but branches out multiple labeled output
  handles (one per `case`/`condition` id, plus `else`) instead of the single
  default handle.
- Edges are a custom removable edge (`RemovableEdge`) — click to reveal a
  delete affordance on the line itself.

All nodes today are the *same generic card* — no per-connector icon/branding
(e.g. Slack, HTTP, Jira would all render identically), no visual cue for node
state (idle vs configured vs invalid), and condition branches only show up as
plain text labels on edges.

**Opening the operation sidebar:** double-clicking a node (or the
"select trigger" placeholder node) slides in a ~500px sidebar
(`PlaybookOperations`) from the left, overlaying the canvas, with a single ×
close button top-right. Its content swaps based on state:

1. **No trigger yet** → `SelectPlaybookTriggerOption`: uppercase title, a
   list of trigger types (manual/webhook) as icon+label option rows fetched
   from the API. Each option's description is a hardcoded placeholder string
   ("Small description about manual") regardless of which trigger it is.
2. **Trigger node selected** → `PlaybookTriggerParameters`: literally an empty
   stub component (`<div>PlaybookTriggerParameters</div>`) — unimplemented.
3. **New/task node, nothing chosen yet** → `SelectTaskOptions`: grouped list
   of task-type option buttons (currently just "connector") with an info
   tooltip per option.
4. **Connector task chosen** → `ConnectorsOption`, a two-step flow:
   - `ConnectorList` — "Step Name" input (oddly disconnected from the form
     below) + a plain list of all available connectors as option rows (no
     icons per connector, no search/category grouping, no descriptions).
   - `ConnectorOperation` — the step-configuration form: Step Name,
     Description, an optional Configuration select, an Operation select, then
     a dynamic parameter list driven by each operation's declared param
     `type`: `text` (plain input), `code` (CodeMirror python editor — a
     commented-out Monaco alternative is left in the file), `cases`/
     `conditions` (the condition-builtin's branch editors — ordered
     if/else-if rows, each with a "route to destination" dropdown, plus an
     `else` row). Ends in a footer with Close/Save.

**Rough edges:**
- Sidebar is a fixed-width overlay, not responsive; no breadcrumbs showing
  "which step am I editing."
- No visual distinction between step types (every connector renders as the
  same generic icon).
- Trigger-parameters screen is entirely unbuilt.
- Condition/branch authoring (`CasesEditor`/`ConditionsEditor`) is functional
  but purely text inputs + a select — no inline preview of the resulting
  branch graph.
- No inline validation/error states shown anywhere in these forms.
- Dead code (commented-out Monaco editor block) still sitting in
  `ConnectorOperation.tsx`.

**Image-gen prompt seed:** "Redesign a visual workflow/playbook builder: a
node-graph canvas with distinct icon-branded nodes per integration (Slack,
HTTP request, condition/branch node with labeled outgoing paths), a slide-in
right-side (or left) inspector panel for configuring the selected step
(tabs for Setup / Parameters / Test), a connector picker with search + icons +
categories, and a branch/condition editor that visually shows each branch
routing to its target node rather than just a dropdown list."

---

## 4. Playbook execution history list — `/playbooks/[playbookId]/history`

**What's there:** Literally `<div>Page</div>` — unimplemented stub.

**Image-gen prompt seed:** "Design a per-playbook run-history list: timeline
or table of past runs for one playbook, each row showing status, trigger
source, duration, started-at, with a link into the run detail/replay view."

---

## 5. Playbook run detail — `/playbooks/[playbookId]/history/[playbookHistoryId]`

**What's there:** Read-only replay of one run on the same canvas
(`ReactFlowPlayground` with `taskHistoryNode`/`edgeHistory` types). Clicking a
node opens a fixed 600px right-side panel with **Output** / **Parameters**
tabs — Output is raw `JSON.stringify` in a `<pre>`, Parameters is an
accordion of raw values per key (also mostly unformatted JSON/strings).

**Rough edges:**
- No run status per node shown on the canvas itself (color-coding for
  success/fail/skipped isn't visually obvious beyond whatever `taskHistoryNode`
  styles — worth checking against the `skipped` status added for conditional
  branching).
- Output/parameters are raw JSON dumps, not formatted/pretty views.
- No re-run / retry-from-here action.

**Image-gen prompt seed:** "Design a workflow run-replay view: the DAG canvas
with each node color-coded by status (success/failed/running/skipped), click
a node to open a detail panel with tabs for Output (pretty-printed,
collapsible JSON tree) and Parameters (rendered as a form-like key/value list,
not raw JSON), plus a retry-from-this-step action."

---

## Screens that don't exist yet (sidebar links to nowhere)

- **Scheduler** — no route, no page.
- **Connectors** (management UI — browse/install/upload connector packages,
  matching the `POST/DELETE /api/connectors/v1` backend) — no route, no page.
  Right now connectors only surface inside the editor's connector picker.

These are good candidates for "design this from scratch" prompts since there's
no existing layout to anchor to — e.g.: "Design a connector marketplace/
management page: grid of installed connectors with icon, name, version,
enabled toggle; an 'Install connector' flow for uploading a zip package; a
detail view showing available operations per connector."
