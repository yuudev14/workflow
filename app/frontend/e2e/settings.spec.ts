import { expect, test, type Browser } from "@playwright/test";
import { adminState, api, newUser, signIn, unique } from "./helpers";

// signInElsewhere signs in with a new browser that has no session.
async function signInElsewhere(browser: Browser, username: string, password: string) {
  const other = await browser.newContext({ storageState: { cookies: [], origins: [] } });
  const page = await other.newPage();
  await page.goto("/login");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  return { page, close: () => other.close() };
}

test.describe("users", () => {
  test("creates a user who can sign in", async ({ page, browser }) => {
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
    await page.getByPlaceholder("Search username or email…").fill(username);
    await expect(page.getByRole("row", { name: new RegExp(username) })).toContainText("analyst");

    const other = await signInElsewhere(browser, username, "password123");
    await expect(other.page).toHaveURL(/\/dashboard/);
    await other.close();
  });

  test("shows an error when the username is in use", async ({ page }) => {
    const user = await newUser(page.request, ["viewer"]);
    await page.goto("/settings/users");
    await page.getByRole("button", { name: "New user" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Username").fill(user.username);
    await dialog.getByLabel("Email").fill(`${unique("other")}@ytsoar.test`);
    await dialog.getByLabel("Password").fill("password123");
    await dialog.getByRole("button", { name: "Create" }).click();
    await expect(page.getByText("Couldn't create the user").first()).toBeVisible();
    await expect(dialog).toBeVisible();
  });

  test("changes the name and the roles of a user", async ({ page }) => {
    const user = await newUser(page.request, ["viewer"]);
    await page.goto("/settings/users");
    await page.getByPlaceholder("Search username or email…").fill(user.username);
    await page.getByRole("row", { name: new RegExp(user.username) }).getByTitle("Edit").click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("First name").fill("Ana");
    await dialog.getByRole("button", { name: "analyst" }).click();
    await dialog.getByRole("button", { name: "Save" }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByRole("row", { name: new RegExp(user.username) })).toContainText("analyst");
  });

  test("resets the password of a user", async ({ page, browser }) => {
    const user = await newUser(page.request, ["viewer"]);
    await page.goto("/settings/users");
    await page.getByPlaceholder("Search username or email…").fill(user.username);
    await page.getByRole("row", { name: new RegExp(user.username) }).getByTitle("Reset password").click();
    await page.getByLabel("New password").fill("password456");
    await page.getByRole("button", { name: "Reset", exact: true }).click();
    await expect(page.getByRole("dialog")).toBeHidden();

    const old = await signInElsewhere(browser, user.username, user.password);
    await expect(old.page.getByRole("alert")).toHaveText("Incorrect username or password.");
    await old.close();
    const fresh = await signInElsewhere(browser, user.username, "password456");
    await expect(fresh.page).toHaveURL(/\/dashboard/);
    await fresh.close();
  });

  test("deactivates a user, who then cannot sign in", async ({ page, browser }) => {
    const user = await newUser(page.request, ["viewer"]);
    await page.goto("/settings/users");
    await page.getByPlaceholder("Search username or email…").fill(user.username);
    await page.getByRole("row", { name: new RegExp(user.username) }).getByTitle("Deactivate").click();
    await page.getByRole("button", { name: "Deactivate" }).click();
    await expect(page.getByRole("row", { name: new RegExp(user.username) })).toContainText("Inactive");

    const other = await signInElsewhere(browser, user.username, user.password);
    await expect(other.page.getByRole("alert")).toHaveText("Incorrect username or password.");
    await other.close();

    await page.getByRole("button", { name: "Inactive", exact: true }).click();
    await page.getByRole("row", { name: new RegExp(user.username) }).getByTitle("Reactivate").click();
    await page.getByRole("button", { name: "Active", exact: true }).click();
    await expect(page.getByRole("row", { name: new RegExp(user.username) })).toContainText("Active");
  });

  test("finds a user by the username", async ({ page }) => {
    const user = await newUser(page.request, ["viewer"]);
    await page.goto("/settings/users");
    await page.getByPlaceholder("Search username or email…").fill(user.username);
    await expect(page.getByRole("row", { name: new RegExp(user.username) })).toBeVisible();
    await expect(page.getByRole("row", { name: /admin@ytsoar.local/ })).toBeHidden();
  });
});

test.describe("roles", () => {
  test("creates a role", async ({ page }) => {
    const name = unique("role");
    await page.goto("/settings/roles");
    await page.getByRole("button", { name: "New role" }).click();
    await page.getByLabel("Name").fill(name);
    await page.getByLabel("Description").fill("made by a browser test");
    await page.getByRole("dialog").getByRole("button", { name: "alerts.read" }).click();
    await page.getByRole("button", { name: "Create role" }).click();
    await expect(page.getByRole("button", { name: new RegExp(name) })).toBeVisible();
  });

  test("shows a builtin role as read-only", async ({ page }) => {
    await page.goto("/settings/roles");
    await page.getByRole("button", { name: /^viewer/ }).click();
    await expect(page.getByRole("button", { name: "alerts.read" })).toBeDisabled();
    await expect(page.getByTitle("Delete role")).toBeHidden();
  });

  test("changes the permissions of a role", async ({ page }) => {
    const name = unique("role");
    const { id } = await api(page.request).post("/roles/v1", { name, permissions: { alerts: ["read"] } });
    await page.goto("/settings/roles");
    await page.getByRole("button", { name: new RegExp(name) }).click();
    await page.getByRole("button", { name: "incidents.read" }).click();
    await page.getByRole("button", { name: "Save permissions" }).click();
    await expect(page.getByText("Permissions updated").first()).toBeVisible();
    await expect.poll(async () => (await api(page.request).get(`/roles/v1/${id}`)).permissions.incidents).toContain("read");
  });

  test("deletes a role", async ({ page }) => {
    const name = unique("role");
    await api(page.request).post("/roles/v1", { name });
    await page.goto("/settings/roles");
    await page.getByRole("button", { name: new RegExp(name) }).click();
    await page.getByTitle("Delete role").click();
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("button", { name: new RegExp(name) })).toBeHidden();
  });
});

test.describe("teams", () => {
  test("creates a team with a role and a member", async ({ page }) => {
    const user = await newUser(page.request, []);
    const name = unique("team");
    await page.goto("/settings/teams");
    await page.getByRole("button", { name: "New team" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name").fill(name);
    await dialog.getByRole("button", { name: "analyst" }).click();
    await dialog.getByPlaceholder("Search users to add…").fill(user.username);
    await dialog.getByRole("checkbox", { name: new RegExp(user.username) }).check();
    await dialog.getByRole("button", { name: "Create" }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(name)).toBeVisible();
  });

  test("changes and deletes a team", async ({ page }) => {
    const name = unique("team");
    await api(page.request).post("/teams/v1", { name });
    await page.goto("/settings/teams");
    await page.getByPlaceholder("Search teams…").fill(name);
    await page.getByRole("row", { name: new RegExp(name) }).getByTitle("Edit").click();
    await page.getByRole("dialog").getByLabel("Description").fill("the blue team");
    await page.getByRole("dialog").getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("the blue team")).toBeVisible();

    await page.getByRole("row", { name: new RegExp(name) }).getByTitle("Delete").click();
    await page.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(page.getByRole("row", { name: new RegExp(name) })).toBeHidden();
  });
});

test.describe("providers", () => {
  test("creates and changes a provider", async ({ page }) => {
    const name = unique("sso");
    await page.goto("/settings/providers");
    await page.getByRole("button", { name: "New provider" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel("Name").fill(name);
    // A disabled provider does not change the login page for the other tests.
    await dialog.getByLabel("Enabled - appears on the login screen").uncheck();
    await dialog.getByRole("button", { name: "Create provider" }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByText(name)).toBeVisible();

    await page.getByRole("row", { name: new RegExp(name) }).getByTitle("Edit").click();
    await dialog.getByLabel("Name").fill(`${name}-changed`);
    await dialog.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText(`${name}-changed`)).toBeVisible();
  });
});

test.describe("audit log", () => {
  test("shows the changes and filters them by module", async ({ page }) => {
    await newUser(page.request, ["viewer"]);
    await page.goto("/settings/audit");
    await expect(page.getByRole("row").nth(1)).toBeVisible();
    await page.getByRole("button", { name: "Auth", exact: true }).click();
    await expect(page.getByRole("row", { name: / settings / })).toHaveCount(0);
    await expect(page.getByRole("row", { name: / auth / }).first()).toBeVisible();
  });
});

test.describe("permissions", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test("lets a viewer read but not change", async ({ page, playwright }) => {
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
    await page.goto("/settings/users");
    await expect(page.getByText("You don't have access to settings")).toBeVisible();
  });
});
