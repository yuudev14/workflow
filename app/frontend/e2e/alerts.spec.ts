import { expect, test } from "@playwright/test";
import { api, newAlert, newPlaybook } from "./helpers";

test.describe("alerts", () => {
  test("the queue lists and finds an alert", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto("/alerts");
    await page.getByPlaceholder("Search alerts…").fill(alert.title);
    await page.getByRole("link", { name: alert.title }).click();
    await expect(page).toHaveURL(`/alerts/${alert.id}`);
    await expect(page.getByRole("heading", { name: alert.title })).toBeVisible();
  });

  test("change the status", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("combobox").first().click();
    await page.getByRole("option", { name: "Investigating" }).click();
    await page.getByRole("button", { name: "Update status" }).click();
    await expect(page.getByRole("button", { name: "Update status" })).toBeHidden();
    expect((await api(page.request).get(`/alerts/v1/${alert.id}`)).status).toBe("investigating");
  });

  test("closing an alert asks for a note", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("combobox").first().click();
    await page.getByRole("option", { name: "Closed" }).click();
    await page.getByPlaceholder("Closure note - why is this being closed?").fill("false alarm");
    await page.getByRole("button", { name: "Update status" }).click();
    await expect(page.getByText("false alarm")).toBeVisible();
  });

  test("add a note", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("tab", { name: /Notes/ }).click();
    await page.getByPlaceholder("Add a note…").fill("checked the host");
    await page.getByRole("button", { name: "Add note" }).click();
    await expect(page.getByText("checked the host")).toBeVisible();
  });

  test("escalate to an incident", async ({ page }) => {
    const alert = await newAlert(page.request);
    await page.goto(`/alerts/${alert.id}`);
    await page.getByRole("button", { name: "Escalate" }).click();
    await expect(page).toHaveURL(/\/incidents\/[0-9a-f-]{36}$/);
    await expect(page.getByRole("link", { name: new RegExp(alert.title) })).toBeVisible();
  });

  test("run a playbook on an alert", async ({ page }) => {
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
      .poll(async () => (await api(page.request).get(`/playbooks/v1/history?playbook_id=${playbook.id}`)).entries[0]?.status, {
        timeout: 30_000,
      })
      .toBe("success");
  });
});
