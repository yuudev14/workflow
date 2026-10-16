import { expect, test } from "@playwright/test";
import { newAlert, newIncident } from "./helpers";

// The date part of the range start, n days before now, as the page writes it.
const daysAgo = (n: number) => {
  const d = new Date(Date.now() - n * 24 * 60 * 60 * 1000);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
};

const dashboards = [
  { path: "/dashboard", heading: "Dashboard", parts: ["Success rate", "Open alerts", "Open incidents"], link: "Open queue", target: "/alerts" },
  { path: "/alerts/dashboard", heading: "Alerts dashboard", parts: ["Alerts by severity", "Alerts by source"], link: "Open queue", target: "/alerts" },
  { path: "/incidents/dashboard", heading: "Incidents dashboard", parts: ["By status", "By severity"], link: "Open cases", target: "/incidents" },
];

for (const dashboard of dashboards) {
  test.describe(dashboard.heading, () => {
    test("shows each part", async ({ page }) => {
      await newAlert(page.request);
      await newIncident(page.request);
      await page.goto(dashboard.path);
      await expect(page.getByRole("heading", { name: dashboard.heading, level: 1 })).toBeVisible();
      for (const part of dashboard.parts) {
        await expect(page.getByText(part).first()).toBeVisible();
      }
    });

    test("changes the date range", async ({ page }) => {
      await page.goto(dashboard.path);
      await expect(page.getByLabel("Range start")).toHaveValue(new RegExp(`^${daysAgo(14)}`));
      await page.getByRole("button", { name: "30d" }).click();
      await expect(page.getByLabel("Range start")).toHaveValue(new RegExp(`^${daysAgo(30)}`));
    });

    test("opens the list", async ({ page }) => {
      await page.goto(dashboard.path);
      await page.getByRole("link", { name: dashboard.link }).click();
      await expect(page).toHaveURL(dashboard.target);
    });
  });
}
