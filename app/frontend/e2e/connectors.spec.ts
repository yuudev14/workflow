import { expect, test } from "@playwright/test";

test("the connectors page lists the installed connectors", async ({ page }) => {
  await page.goto("/connectors");
  await expect(page.getByRole("heading", { name: "Connectors" })).toBeVisible();
  for (const name of ["Condition", "HTTP Request", "Sample JS"]) {
    await expect(page.getByText(name, { exact: true }).first()).toBeVisible();
  }
});
