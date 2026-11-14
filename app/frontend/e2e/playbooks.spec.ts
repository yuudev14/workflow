import { expect, test, type Page } from "@playwright/test";
import { api, newPlaybook, unique } from "./helpers";

type Task = { name: string; parameters: Record<string, unknown> };
const tasksOf = async (page: Page, id: string): Promise<Task[]> =>
  (await api(page.request).get(`/playbooks/${id}/tasks`)).tasks;

// The editor toolbar lies over the canvas, so the tests send the events to the
// node and do not click at its position.
const node = (page: Page, name: string) => page.locator(".react-flow__node").filter({ hasText: name });

test.describe("playbook list", () => {
  test("creates a playbook and opens it", async ({ page }) => {
    const name = unique("playbook");
    await page.goto("/playbooks");
    await page.getByRole("button", { name: "Create playbook" }).click();
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Description").fill("made by a browser test");
    await page.getByRole("button", { name: "Submit" }).click();

    await expect(page).toHaveURL(/\/playbooks\/[0-9a-f-]{36}$/);
    await expect(page.getByRole("heading", { name })).toBeVisible();
  });

  test("shows an error when the name is missing", async ({ page }) => {
    await page.goto("/playbooks");
    await page.getByRole("button", { name: "Create playbook" }).click();
    await page.getByRole("button", { name: "Submit" }).click();
    await expect(page.getByText("name must be at least 2 characters.")).toBeVisible();
  });

  test("finds a playbook by its name", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    const other = await newPlaybook(page.request);
    await page.goto("/playbooks");
    await page.getByPlaceholder("Search playbooks…").fill(playbook.name);
    await expect(page.getByRole("link", { name: new RegExp(playbook.name) })).toBeVisible();
    await expect(page.getByRole("link", { name: new RegExp(other.name) })).toBeHidden();
  });

  test("keeps the columns aligned when descriptions differ in length", async ({ page }) => {
    const short = await newPlaybook(page.request);
    const long = await newPlaybook(page.request);
    await api(page.request).put(`/playbooks/${long.id}`, { description: "long description ".repeat(30) });
    await page.goto("/playbooks");
    const badgeX = async (name: string) =>
      (await page.getByRole("link", { name: new RegExp(name) }).getByText("Manual").boundingBox())?.x;
    expect(await badgeX(short.name)).toBe(await badgeX(long.name));
  });
});

test.describe("playbook editor", () => {
  test("deletes the playbook and opens the list", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    // The list loads first, so the delete must refresh a list that is cached and not on screen.
    await page.goto("/playbooks");
    await page.getByRole("link", { name: new RegExp(playbook.name) }).click();
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();

    await expect(page).toHaveURL(/\/playbooks$/);
    await expect(page.getByRole("link", { name: new RegExp(playbook.name) })).toBeHidden();
  });

  test("draws the branch label outside the condition node when the target is on its right", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    const expression = "{{ var.input.records[0].severity == 'critical' }}";
    await api(page.request).put(`/playbooks/tasks/${playbook.id}`, {
      nodes: [
        { name: "start", operation: "start", parameters: {}, x: 60, y: 100 },
        {
          name: "gate",
          operation: "switch_expression",
          connector_id: "condition",
          connector_name: "Condition",
          parameters: { cases: [{ id: "yes", expression }] },
          x: 440,
          y: 100,
        },
        {
          name: "hello",
          operation: "python_code",
          connector_id: "code_snippet_py",
          connector_name: "Code Snippet",
          parameters: { code: "print('hello')" },
          x: 820,
          y: 100,
        },
      ],
      edges: { start: ["gate"], gate: ["hello"] },
      handles: {
        start: { gate: { source_handle: "source-right", destination_handle: "target-left" } },
        gate: { hello: { source_handle: "yes", destination_handle: "target-left" } },
      },
    });
    await page.goto(`/playbooks/${playbook.id}`);

    const label = await page.getByTitle(expression).boundingBox();
    const gate = await node(page, "gate").boundingBox();
    expect(label!.x).toBeGreaterThanOrEqual(gate!.x + gate!.width);
  });

  test("shows the steps", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto(`/playbooks/${playbook.id}`);
    await expect(page.getByRole("heading", { name: playbook.name })).toBeVisible();
    await expect(node(page, "hello")).toBeVisible();
  });

  test("changes a step and saves the playbook", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto(`/playbooks/${playbook.id}`);
    await node(page, "hello").dispatchEvent("dblclick");
    await page.getByLabel("Step name").fill("renamed");
    await page.getByRole("button", { name: "Save step" }).click();
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect.poll(async () => (await tasksOf(page, playbook.id)).map((t) => t.name)).toContain("renamed");
  });

  test("adds a Python step after a step", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto(`/playbooks/${playbook.id}`);
    // Drag a connection from the step to an empty place on the canvas.
    const handle = (await node(page, "hello").locator(".react-flow__handle.source.react-flow__handle-right").boundingBox())!;
    await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2);
    await page.mouse.down();
    await page.mouse.move(handle.x + 250, handle.y + 120, { steps: 15 });
    await page.mouse.up();

    await page.getByText("Code snippet").first().click();
    await page.getByLabel("Step name").fill("added");
    await page.locator(".cm-content").fill("result = 'added'");
    await page.getByRole("button", { name: "Save step" }).click();
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect.poll(async () => (await tasksOf(page, playbook.id)).map((t) => t.name)).toContain("added");
    const added = (await tasksOf(page, playbook.id)).find((t) => t.name === "added")!;
    expect(added.parameters.code).toBe("result = 'added'");
  });

  test("deletes a step", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    const step = (name: string) => ({ name, operation: "python_code", connector_id: "code_snippet_py", connector_name: "Code Snippet", parameters: { code: "" } });
    await api(page.request).put(`/playbooks/tasks/${playbook.id}`, {
      nodes: [{ name: "start", operation: "start", parameters: {} }, step("first"), step("second")],
      edges: { start: ["first"], first: ["second"] },
    });
    await page.goto(`/playbooks/${playbook.id}`);
    await node(page, "second").getByRole("button", { name: "Delete Node" }).dispatchEvent("click");
    await expect(node(page, "second")).toBeHidden();
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect.poll(async () => (await tasksOf(page, playbook.id)).map((t) => t.name).sort()).toEqual(["first", "start"]);
  });

  test("inserts templates of earlier steps and the run input from the variables panel", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    const http = (name: string) => ({ name, operation: "get_request", connector_id: "http_request", connector_name: "HTTP Request", parameters: { url: "" } });
    await api(page.request).put(`/playbooks/tasks/${playbook.id}`, {
      nodes: [
        { name: "start", operation: "start", parameters: {} },
        http("lookup"),
        http("fetch"),
        { name: "check", operation: "python_code", connector_id: "code_snippet_py", connector_name: "Code Snippet", parameters: { code: "" } },
      ],
      edges: { start: ["lookup"], lookup: ["fetch"], fetch: ["check"] },
    });
    await page.goto(`/playbooks/${playbook.id}`);

    await node(page, "fetch").dispatchEvent("dblclick");
    const variables = page.getByRole("complementary", { name: "Variables" });
    await page.getByPlaceholder("https://api.example.com/items").focus();
    await variables.getByRole("button", { name: "Insert var.steps.lookup.body" }).click();
    await page.getByRole("button", { name: "Save step" }).click();

    await node(page, "check").dispatchEvent("dblclick");
    await page.locator(".cm-content").click();
    await variables.getByRole("button", { name: "Insert var.steps.fetch", exact: true }).click();
    await variables.getByPlaceholder("Search variables").fill("title");
    await variables.getByRole("button", { name: "Insert var.input.records[0].title" }).click();
    await page.getByRole("button", { name: "Save step" }).click();
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect
      .poll(async () => (await tasksOf(page, playbook.id)).find((t) => t.name === "check")?.parameters.code)
      .toBe("{{ var.steps.fetch }}{{ var.input.records[0].title }}");
    const fetch = (await tasksOf(page, playbook.id)).find((t) => t.name === "fetch")!;
    expect(fetch.parameters.url).toBe("{{ var.steps.lookup.body }}");
  });

  test("shows the webhook URL and saves the auth type", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await api(page.request).put(`/playbooks/${playbook.id}`, { trigger_type: "webhook" });
    await page.goto(`/playbooks/${playbook.id}`);
    await node(page, "Trigger").dispatchEvent("dblclick");
    await expect(page.getByText(`/api/v1/webhooks/${playbook.id}`).first()).toBeVisible();
    await expect(page.getByText("Authorization: Bearer <API key>")).toBeVisible();

    await page.getByLabel("Authentication").click();
    await page.getByRole("option", { name: "None" }).click();
    await expect(page.getByText("Anyone with this URL can run this playbook.")).toBeVisible();
    await page.getByRole("button", { name: "Save trigger" }).click();
    await page.getByRole("button", { name: "Save", exact: true }).click();

    await expect
      .poll(async () => (await api(page.request).get(`/playbooks/${playbook.id}`)).trigger_parameters)
      .toEqual({ auth: "none" });
  });
});

test.describe("runs", () => {
  test("shows success in the history", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto(`/playbooks/${playbook.id}`);
    await page.getByRole("button", { name: "Trigger" }).click();
    await expect(page.getByText("Playbook triggered").first()).toBeVisible();

    // The worker runs the playbook, and the page gets the status on the WebSocket.
    await page.getByRole("link", { name: "Runs" }).click();
    await expect(page.getByRole("link", { name: /Success/ })).toBeVisible({ timeout: 30_000 });
  });

  test("shows the output of a step", async ({ page }) => {
    const playbook = await newPlaybook(page.request, unique("playbook"), "result = 'from the sandbox'");
    const run = await api(page.request).post(`/playbooks/trigger/${playbook.id}`);
    await page.goto(`/playbooks/${playbook.id}/history/${run.playbook_history_id}`);
    await expect(page.getByRole("link", { name: /Success/ })).toBeVisible({ timeout: 30_000 });
    await page.reload();
    await node(page, "hello").dispatchEvent("dblclick");
    await expect(page.getByRole("complementary")).toContainText("from the sandbox");
  });

  test("shows failed when a step fails", async ({ page }) => {
    const playbook = await newPlaybook(page.request, unique("playbook"), "raise Exception('boom')");
    await api(page.request).post(`/playbooks/trigger/${playbook.id}`);
    await page.goto(`/playbooks/${playbook.id}/history`);
    await expect(page.getByRole("link", { name: /Failed/ })).toBeVisible({ timeout: 30_000 });
  });

  test("filters the runs in the history window", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await api(page.request).post(`/playbooks/trigger/${playbook.id}`);
    await page.goto(`/playbooks/${playbook.id}`);
    await page.getByRole("button", { name: "History" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("button", { name: /\d Success$/ })).toBeVisible({ timeout: 30_000 });
    await dialog.getByRole("button", { name: "Failed", exact: true }).click();
    await expect(dialog.getByText("0 runs")).toBeVisible();
  });

  test("keeps the editor flow after the history window closes", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await api(page.request).post(`/playbooks/trigger/${playbook.id}`);
    await page.goto(`/playbooks/${playbook.id}`);
    await expect(node(page, "hello")).toBeVisible();
    await page.getByRole("button", { name: "History" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("button", { name: /\d Success$/ })).toBeVisible({ timeout: 30_000 });
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(node(page, "hello")).toBeVisible();
  });

  test("lists the runs on the executions page", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await api(page.request).post(`/playbooks/trigger/${playbook.id}`);
    await page.goto("/playbooks/executions");
    await expect(page.getByRole("link", { name: new RegExp(playbook.name) })).toBeVisible();
  });
});
