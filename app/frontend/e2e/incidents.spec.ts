import { expect, test } from "@playwright/test";
import { api, newAlert, newIncident, unique } from "./helpers";

test.describe("incident list", () => {
  test("creates an incident", async ({ page }) => {
    const title = unique("incident");
    await page.goto("/incidents");
    await page.getByRole("button", { name: "New incident" }).click();
    await expect(page.getByRole("button", { name: "Create incident" })).toBeDisabled();
    await page.getByLabel("Title").fill(title);
    await page.getByLabel("Severity").click();
    await page.getByRole("option", { name: "Critical" }).click();
    await page.getByRole("button", { name: "Create incident" }).click();

    await page.getByPlaceholder("Search incidents…").fill(title);
    await expect(page.getByRole("link", { name: title })).toBeVisible();
  });

  test("shows the open or the resolved incidents", async ({ page }) => {
    const open = await newIncident(page.request);
    const resolved = await newIncident(page.request);
    await api(page.request).patch(`/incidents/${resolved.id}/status`, { status: "resolved" });
    await page.goto("/incidents");
    await page.getByRole("button", { name: "Resolved", exact: true }).click();
    await page.getByPlaceholder("Search incidents…").fill(resolved.title);
    await expect(page.getByRole("link", { name: resolved.title })).toBeVisible();
    await page.getByPlaceholder("Search incidents…").fill(open.title);
    await expect(page.getByRole("link", { name: open.title })).toBeHidden();
    await page.getByRole("button", { name: "Open", exact: true }).click();
    await expect(page.getByRole("link", { name: open.title })).toBeVisible();
  });
});

test.describe("incident detail", () => {
  test("marks the incident contained", async ({ page }) => {
    const incident = await newIncident(page.request);
    await page.goto(`/incidents/${incident.id}`);
    await page.getByRole("button", { name: "Mark contained" }).click();
    await expect.poll(async () => (await api(page.request).get(`/incidents/${incident.id}`)).status).toBe("contained");
  });

  test("adds a note", async ({ page }) => {
    const incident = await newIncident(page.request);
    await page.goto(`/incidents/${incident.id}`);
    await page.getByRole("tab", { name: /Notes/ }).click();
    await page.getByPlaceholder("Add a note…").fill("isolated the host");
    await page.getByRole("button", { name: "Add note" }).click();
    await expect(page.getByText("isolated the host")).toBeVisible();
  });

  test("unlinks an alert", async ({ page }) => {
    const incident = await newIncident(page.request);
    const alert = await newAlert(page.request);
    await api(page.request).post(`/incidents/${incident.id}/alerts`, { alert_id: alert.id });
    await page.goto(`/incidents/${incident.id}`);
    await page.getByRole("tab", { name: /Related/ }).click();
    await expect(page.getByText(alert.title)).toBeVisible();
    await page.getByRole("button", { name: "Unlink alert" }).click();
    await expect(page.getByText(alert.title)).toBeHidden();
  });
});
