import { expect, test } from "@playwright/test";
import { api, newAlert, unique } from "./helpers";

async function newIncident(page: import("@playwright/test").Page) {
  const title = unique("incident");
  const { id } = await api(page.request).post("/incidents/v1", { title, severity: "medium" });
  return { id: id as string, title };
}

test.describe("incidents", () => {
  test("create an incident", async ({ page }) => {
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

  test("mark an incident contained", async ({ page }) => {
    const incident = await newIncident(page);
    await page.goto(`/incidents/${incident.id}`);
    await page.getByRole("button", { name: "Mark contained" }).click();
    await expect.poll(async () => (await api(page.request).get(`/incidents/v1/${incident.id}`)).status).toBe("contained");
  });

  test("add a note", async ({ page }) => {
    const incident = await newIncident(page);
    await page.goto(`/incidents/${incident.id}`);
    await page.getByRole("tab", { name: /Notes/ }).click();
    await page.getByPlaceholder("Add a note…").fill("isolated the host");
    await page.getByRole("button", { name: "Add note" }).click();
    await expect(page.getByText("isolated the host")).toBeVisible();
  });

  test("unlink an alert", async ({ page }) => {
    const incident = await newIncident(page);
    const alert = await newAlert(page.request);
    await api(page.request).post(`/incidents/v1/${incident.id}/alerts`, { alert_id: alert.id });
    await page.goto(`/incidents/${incident.id}`);
    await page.getByRole("tab", { name: /Related/ }).click();
    await expect(page.getByText(alert.title)).toBeVisible();
    await page.getByRole("button", { name: "Unlink alert" }).click();
    await expect(page.getByText(alert.title)).toBeHidden();
  });
});
