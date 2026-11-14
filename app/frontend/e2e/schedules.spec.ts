import { expect, test } from "@playwright/test";
import { newPlaybook, unique } from "./helpers";

test.describe("schedules", () => {
  test("creates, edits, disables and deletes a schedule", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    const name = unique("nightly");
    await page.goto("/schedules");
    await page.getByRole("button", { name: "New schedule" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name").fill(name);
    await dialog.getByLabel("Playbook").click();
    await page.getByRole("option", { name: playbook.name }).click();
    await dialog.getByLabel("Cron").fill("0 2 * * *");
    await expect(dialog.getByText("Will run at 02:00 (UTC)")).toBeVisible();
    await expect(dialog.getByText("Next runs")).toBeVisible();
    await dialog.getByRole("button", { name: "Create" }).click();
    await expect(dialog).toBeHidden();

    const row = page.getByRole("row", { name: new RegExp(name) });
    await expect(row).toContainText("0 2 * * *");
    await expect(row).toContainText(playbook.name);

    await row.getByTitle("Edit").click();
    await dialog.getByLabel("Cron").fill("30 3 * * 1");
    await dialog.getByRole("button", { name: "Save" }).click();
    await expect(dialog).toBeHidden();
    await expect(row).toContainText("30 3 * * 1");

    await row.getByRole("switch").click();
    await expect(row.getByRole("switch")).toHaveText("Off");

    await row.getByTitle("Delete").click();
    await page.getByRole("dialog").getByRole("button", { name: "Delete" }).click();
    await expect(row).toBeHidden();
  });

  test("shows an error when the cron is not valid", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    await page.goto("/schedules");
    await page.getByRole("button", { name: "New schedule" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name").fill(unique("bad"));
    await dialog.getByLabel("Playbook").click();
    await page.getByRole("option", { name: playbook.name }).click();
    await dialog.getByLabel("Cron").fill("@daily");
    await expect(dialog.getByText(/cron must have 5 fields/)).toBeVisible();
    await dialog.getByRole("button", { name: "Create" }).click();
    await expect(page.getByText("Couldn't save the schedule").first()).toBeVisible();
    await expect(dialog).toBeVisible();
  });

  test("lists the schedules of a playbook in the editor", async ({ page }) => {
    const playbook = await newPlaybook(page.request);
    const name = unique("hourly");
    await page.request.post("/playbook-api/api/v1/schedules", { data: { name, playbook_id: playbook.id, cron: "0 * * * *" } });
    await page.goto(`/playbooks/${playbook.id}`);
    await page.getByRole("button", { name: "Schedules" }).click();
    await expect(page.getByRole("dialog")).toContainText(name);
    await page.getByRole("link", { name: "Open the Schedules page" }).click();
    await expect(page).toHaveURL(/\/schedules$/);
  });
});
