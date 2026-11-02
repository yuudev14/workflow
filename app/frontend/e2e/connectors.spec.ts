import { expect, test } from "@playwright/test";
import { unique } from "./helpers";

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

test("adds, edits and deletes a connector configuration", async ({ page }) => {
  const name = unique("prod");
  await page.goto("/connectors");
  await page.getByRole("button", { name: /^Sample\b/ }).first().click();
  const sheet = page.getByRole("dialog");
  await sheet.getByRole("button", { name: "New configuration" }).click();

  const form = page.getByRole("dialog", { name: "New configuration" });
  await form.getByLabel("Name").fill(name);
  await form.getByLabel("API Key").fill("secret-key");
  await form.getByRole("button", { name: "Create configuration" }).click();
  await expect(page.getByText("Configuration created").first()).toBeVisible();
  await expect(sheet.getByText(name)).toBeVisible();

  await sheet.getByTitle(`Edit ${name}`).click();
  const edit = page.getByRole("dialog", { name: `Edit ${name}` });
  await expect(edit.getByLabel("API Key")).toHaveValue("********");
  await edit.getByRole("button", { name: "Cancel" }).click();

  await sheet.getByTitle(`Delete ${name}`).click();
  await page.getByRole("dialog", { name: "Delete the configuration?" }).getByRole("button", { name: "Delete" }).click();
  await expect(sheet.getByText(name)).toBeHidden();
});
