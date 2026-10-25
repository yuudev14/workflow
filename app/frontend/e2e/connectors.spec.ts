import { expect, test } from "@playwright/test";

test("the connectors page lists the installed connectors", async ({ page }) => {
  await page.goto("/connectors");
  await expect(page.getByRole("heading", { name: "Connectors" })).toBeVisible();
  for (const name of ["Condition", "HTTP Request", "Sample JS"]) {
    await expect(page.getByText(name, { exact: true }).first()).toBeVisible();
  }
});

// The api installs into the connectors of the repository, so the tests do not
// install a new connector or replace one.
test.describe("connector upload", () => {
  test("asks before it replaces an installed connector", async ({ page }) => {
    await page.goto("/connectors");
    await page.getByLabel("Connector zip").setInputFiles("e2e/fixtures/sample_js.zip");
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByText("Replace the connector?")).toBeVisible();
    await expect(dialog).toContainText("sample_js");
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(dialog).toBeHidden();
  });

  test("shows an error when the file is not a connector zip", async ({ page }) => {
    await page.goto("/connectors");
    await page.getByLabel("Connector zip").setInputFiles({ name: "bad.zip", mimeType: "application/zip", buffer: Buffer.from("not a zip") });
    await expect(page.getByText("Couldn't install the connector").first()).toBeVisible();
  });
});
