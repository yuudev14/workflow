import { expect, test } from "@playwright/test";
import { api, newPlaybook, unique } from "./helpers";

test.describe("playbooks", () => {
  test("create a playbook", async ({ page }) => {
    const name = unique("playbook");
    await page.goto("/playbooks");
    await page.getByRole("button", { name: "Create playbook" }).click();
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Description").fill("made by a browser test");
    await page.getByRole("button", { name: "Submit" }).click();

    await expect(page).toHaveURL(/\/playbooks\/[0-9a-f-]{36}$/);
    await expect(page.getByRole("heading", { name })).toBeVisible();
    await page.goto("/playbooks");
    await page.getByPlaceholder("Search playbooks…").fill(name);
    await expect(page.getByRole("link", { name: new RegExp(name) })).toBeVisible();
  });

  test("a playbook needs a name", async ({ page }) => {
    await page.goto("/playbooks");
    await page.getByRole("button", { name: "Create playbook" }).click();
    await page.getByRole("button", { name: "Submit" }).click();
    await expect(page.getByText("name must be at least 2 characters.")).toBeVisible();
  });

  test("the editor shows the steps", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto(`/playbooks/${playbook.id}`);
    await expect(page.getByRole("heading", { name: playbook.name })).toBeVisible();
    await expect(page.getByText("hello", { exact: true })).toBeVisible();
  });
});

test.describe("runs", () => {
  test("a run shows success in the history", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto(`/playbooks/${playbook.id}`);
    await page.getByRole("button", { name: "Trigger" }).click();
    await expect(page.getByText("Playbook triggered").first()).toBeVisible();

    // The worker runs the playbook, and the page gets the status on the WebSocket.
    await page.getByRole("link", { name: "Runs" }).click();
    await expect(page.getByRole("link", { name: /Success/ })).toBeVisible({ timeout: 30_000 });
    await page.getByRole("link", { name: /Success/ }).click();
    await expect(page.getByText("hello", { exact: true })).toBeVisible();
  });

  test("a step that fails makes the run fail", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await api(page.request).put(`/playbooks/v1/tasks/${playbook.id}`, {
      nodes: [
        { name: "start", operation: "start", parameters: {} },
        {
          name: "boom",
          operation: "python_code",
          connector_id: "code_snippet_py",
          connector_name: "Code Snippet",
          parameters: { code: "raise Exception('boom')" },
        },
      ],
      edges: { start: ["boom"] },
    });
    await api(page.request).post(`/playbooks/v1/trigger/${playbook.id}`);

    await page.goto(`/playbooks/${playbook.id}/history`);
    await expect(page.getByRole("link", { name: /Failed/ })).toBeVisible({ timeout: 30_000 });
  });

  test("the executions page lists the runs", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await api(page.request).post(`/playbooks/v1/trigger/${playbook.id}`);
    await page.goto("/playbooks/executions");
    await expect(page.getByRole("link", { name: new RegExp(playbook.name) })).toBeVisible();
  });
});
