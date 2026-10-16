import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { adminState } from "./helpers";

// These tests sign in through Keycloak, as a real user does. Keycloak must run
// at localhost:8180 with the realm in docker/keycloak.
const keycloak = "http://localhost:8180/realms/ytsoar";

// The provider row holds the client secret, so it is not a migration. The test
// makes it, the same as the admin does in Settings.
async function addKeycloakProvider(request: APIRequestContext) {
  const res = await request.post("/playbook-api/api/auth-providers/v1", {
    data: {
      type: "oidc",
      name: "Keycloak",
      enabled: true,
      config: {
        issuer: keycloak,
        internal_issuer: keycloak,
        client_id: "ytsoar",
        client_secret: "ytsoar-dev-secret",
        scopes: ["openid", "profile", "email"],
        groups_claim: "groups",
        group_role_mapping: { "soc-admins": "admin", "soc-analysts": "analyst" },
        default_role: "viewer",
        allow_jit: true,
      },
    },
  });
  // 409: an earlier run made it.
  expect([201, 409]).toContain(res.status());
}

async function signInWithKeycloak(page: Page, username: string, password: string) {
  await page.goto("/login");
  await page.getByRole("button", { name: "Continue with Keycloak" }).click();
  await expect(page).toHaveURL(/localhost:8180/);
  await page.getByLabel("Username or email").fill(username);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Sign In" }).click();
  await expect(page).toHaveURL(/localhost:3000\/dashboard/);
}

test.describe("keycloak", () => {
  test.beforeAll(async ({ playwright }) => {
    const admin = await playwright.request.newContext({ baseURL: "http://localhost:3000", storageState: adminState });
    await addKeycloakProvider(admin);
    await admin.dispose();
  });

  test.use({ storageState: { cookies: [], origins: [] } });

  test("a Keycloak admin gets the admin role", async ({ page }) => {
    await signInWithKeycloak(page, "kc-admin", "admin123");
    await page.goto("/settings/users");
    await expect(page.getByRole("heading", { name: "Users" })).toBeVisible();
    await expect(page.getByRole("row", { name: /kc-admin/ })).toContainText("admin");
  });

  test("a Keycloak analyst gets the analyst role", async ({ page }) => {
    await signInWithKeycloak(page, "kc-analyst", "analyst123");
    await page.goto("/alerts");
    await expect(page.getByRole("heading", { name: "Alerts" })).toBeVisible();
    await page.getByRole("button", { name: /kc-analyst/ }).first().click();
    await expect(page.getByRole("menuitem", { name: "Settings" })).toBeHidden();
  });

  test("a wrong Keycloak password does not sign in", async ({ page }) => {
    await page.goto("/login");
    await page.getByRole("button", { name: "Continue with Keycloak" }).click();
    await page.getByLabel("Username or email").fill("kc-analyst");
    await page.getByLabel("Password", { exact: true }).fill("wrong");
    await page.getByRole("button", { name: "Sign In" }).click();
    await expect(page.getByText("Invalid username or password.")).toBeVisible();
  });
});
