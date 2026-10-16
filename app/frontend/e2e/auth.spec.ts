import { expect, test } from "@playwright/test";
import { admin, signIn } from "./helpers";

test.describe("sign in", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("a page needs a session", async ({ page }) => {
    await page.goto("/alerts");
    await expect(page).toHaveURL(/\/login/);
  });

  test("a wrong password shows an error", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Username").fill(admin.username);
    await page.getByLabel("Password").fill("wrong");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("alert")).toHaveText("Incorrect username or password.");
    await expect(page).toHaveURL(/\/login/);
  });

  test("empty fields show errors", async ({ page }) => {
    await page.goto("/login");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByText("Username is required")).toBeVisible();
    await expect(page.getByText("Password is required")).toBeVisible();
  });

  test("the correct password opens the dashboard", async ({ page }) => {
    await signIn(page, admin.username, admin.password);
    await expect(page.getByText("admin@ytsoar.local").first()).toBeVisible();
  });

  test("log out ends the session", async ({ page }) => {
    await signIn(page, admin.username, admin.password);
    await page.getByRole("button", { name: /admin@ytsoar\.local/ }).first().click();
    await page.getByRole("menuitem", { name: "Log out" }).click();
    await expect(page).toHaveURL(/\/login/);
    await page.goto("/alerts");
    await expect(page).toHaveURL(/\/login/);
  });
});
