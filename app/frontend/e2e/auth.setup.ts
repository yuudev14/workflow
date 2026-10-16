import { test as setup } from "@playwright/test";
import { admin, adminState, signIn } from "./helpers";

setup("sign in as the admin", async ({ page }) => {
  await signIn(page, admin.username, admin.password);
  await page.context().storageState({ path: adminState });
});
