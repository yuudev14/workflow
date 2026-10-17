import { expect, test } from "@playwright/test";
import { api, newAlert, newPlaybook, unique } from "./helpers";

test.describe("alert queue", () => {
  test("finds an alert and opens it", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto("/alerts");
    await page.getByPlaceholder("Search alerts…").fill(alert.title);
    await page.getByRole("link", { name: alert.title }).click();
    await expect(page).toHaveURL(`/alerts/${alert.id}`);
    await expect(page.getByRole("heading", { name: alert.title })).toBeVisible();
  });

  test("filters by severity", async ({ page }) => {
    const high = await newAlert(page.request);
    const { id } = await api(page.request).post("/alerts", { title: unique("low"), severity: "low", source_kind: "edr" });
    await page.goto("/alerts");
    await page.getByRole("button", { name: "Severity" }).click();
    await page.getByRole("menuitemcheckbox", { name: /^Low/ }).click();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("link", { name: high.title })).toBeHidden();
    const low = (await api(page.request).get(`/alerts/${id}`)).title;
    await expect(page.getByRole("link", { name: low })).toBeVisible();
  });

  test("shows no rows when the search finds nothing", async ({ page }) => {
    await page.goto("/alerts");
    await page.getByPlaceholder("Search alerts…").fill(unique("nothing"));
    await expect(page.getByRole("row", { name: /Select row/ })).toHaveCount(0);
  });

  test("runs a playbook on the selected alerts", async ({ page }) => {
    const alert = await newAlert(page.request);
    const playbook = await newPlaybook(page.request);
    await page.goto("/alerts");
    await page.getByPlaceholder("Search alerts…").fill(alert.title);
    await page.getByRole("row", { name: new RegExp(alert.title) }).getByRole("checkbox").check();
    await expect(page.getByText("1 selected")).toBeVisible();
    await page.getByRole("button", { name: "Run playbook" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("combobox").click();
    await page.getByRole("option", { name: playbook.name }).click();
    await dialog.getByRole("button", { name: "Run" }).click();
    await expect(dialog).toBeHidden();
    await expect
      .poll(async () => (await api(page.request).get(`/playbooks/history?playbook_id=${playbook.id}`)).entries.length)
      .toBe(1);
  });
});

test.describe("alert detail", () => {
  test("changes the status", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("combobox").first().click();
    await page.getByRole("option", { name: "Investigating" }).click();
    await page.getByRole("button", { name: "Update status" }).click();
    await expect(page.getByRole("button", { name: "Update status" })).toBeHidden();
    expect((await api(page.request).get(`/alerts/${alert.id}`)).status).toBe("investigating");
  });

  test("asks for a note when the alert closes", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("combobox").first().click();
    await page.getByRole("option", { name: "Closed" }).click();
    await page.getByPlaceholder("Closure note - why is this being closed?").fill("false alarm");
    await page.getByRole("button", { name: "Update status" }).click();
    await expect(page.getByText("false alarm")).toBeVisible();
  });

  test("adds a note", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("tab", { name: /Notes/ }).click();
    await page.getByPlaceholder("Add a note…").fill("checked the host");
    await page.getByRole("button", { name: "Add note" }).click();
    await expect(page.getByText("checked the host")).toBeVisible();
  });

  test("escalates the alert to an incident", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("button", { name: "Escalate" }).click();
    await expect(page).toHaveURL(/\/incidents\/[0-9a-f-]{36}$/);
    await page.getByRole("tab", { name: /Related/ }).click();
    await expect(page.getByRole("link", { name: new RegExp(alert.title) })).toBeVisible();
  });

  test("runs a playbook on the alert", async ({ page }) => {
    const alert = await newAlert(page.request);
    const playbook = await newPlaybook(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("button", { name: "Run playbook" }).first().click();
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("combobox").click();
    await page.getByRole("option", { name: playbook.name }).click();
    await dialog.getByRole("button", { name: "Run" }).click();
    await expect(dialog).toBeHidden();

    await expect
      .poll(async () => (await api(page.request).get(`/playbooks/history?playbook_id=${playbook.id}`)).entries[0]?.status, {
        timeout: 30_000,
      })
      .toBe("success");
  });
});
