import { expect, test } from "@playwright/test";
import { adminState, newUser, signIn, unique } from "./helpers";

test.describe("users", () => {
  test("create a user who can sign in", async ({ page, browser }) => {
    const username = unique("user");
    await page.goto("/settings/users");
    await page.getByRole("button", { name: "New user" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Username").fill(username);
    await dialog.getByLabel("Email").fill(`${username}@ytsoar.test`);
    await dialog.getByLabel("Password").fill("password123");
    await dialog.getByRole("button", { name: "analyst" }).click();
    await dialog.getByRole("button", { name: "Create" }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByRole("row", { name: new RegExp(username) })).toContainText("analyst");

    const other = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    await signIn(await other.newPage(), username, "password123");
    await other.close();
  });

  test("a deactivated user cannot sign in", async ({ page, browser }) => {
    const user = await newUser(page.request, ["viewer"]);
    await page.goto("/settings/users");
    await page.getByRole("row", { name: new RegExp(user.username) }).getByTitle("Deactivate").click();
    await page.getByRole("button", { name: "Deactivate" }).click();
    await expect(page.getByRole("row", { name: new RegExp(user.username) })).toContainText("Inactive");

    const other = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const otherPage = await other.newPage();
    await otherPage.goto("/login");
    await otherPage.getByLabel("Username").fill(user.username);
    await otherPage.getByLabel("Password").fill(user.password);
    await otherPage.getByRole("button", { name: "Sign in" }).click();
    await expect(otherPage.getByRole("alert")).toHaveText("Incorrect username or password.");
    await other.close();
  });
});

test.describe("roles and teams", () => {
  test("create a role", async ({ page }) => {
    const name = unique("role");
    await page.goto("/settings/roles");
    await page.getByRole("button", { name: "New role" }).click();
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Description").fill("made by a browser test");
    await page.getByRole("button", { name: "Create role" }).click();
    await expect(page.getByRole("button", { name: new RegExp(name) })).toBeVisible();
  });

  test("create a team", async ({ page }) => {
    const name = unique("team");
    await page.goto("/settings/teams");
    await page.getByRole("button", { name: "New team" }).click();
    await page.getByRole("dialog").getByLabel("Name").fill(name);
    await page.getByRole("dialog").getByRole("button", { name: "Create" }).click();
    await expect(page.getByText(name)).toBeVisible();
  });

  test("the audit log shows the changes", async ({ page }) => {
    await newUser(page.request, ["viewer"]);
    await page.goto("/settings/audit");
    await expect(page.getByRole("row").nth(1)).toBeVisible();
  });
});

test.describe("permissions", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("a viewer can read but cannot change", async ({ page, playwright }) => {
    const adminRequest = await playwright.request.newContext({ baseURL: "http://localhost:3000", storageState: adminState });
    const viewer = await newUser(adminRequest, ["viewer"]);
    await adminRequest.dispose();

    await signIn(page, viewer.username, viewer.password);
    await page.goto("/playbooks");
    await expect(page.getByRole("heading", { name: "Playbooks" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Create playbook" })).toBeHidden();
    await page.goto("/incidents");
    await expect(page.getByRole("button", { name: "New incident" })).toBeHidden();
    await page.getByRole("button", { name: new RegExp(viewer.username) }).first().click();
    await expect(page.getByRole("menuitem", { name: "Settings" })).toBeHidden();
  });
});
