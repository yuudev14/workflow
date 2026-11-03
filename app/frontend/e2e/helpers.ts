import { expect, type APIRequestContext, type Page } from "@playwright/test";

export const admin = { username: "admin", password: "admin123!" };
export const adminState = "e2e/.auth/admin.json";

export const unique = (name: string) => `${name}-${crypto.randomUUID().slice(0, 8)}`;

export async function signIn(page: Page, username: string, password: string) {
  await page.goto("/login");
  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(/\/dashboard/);
}

// api sends requests with the session cookie of the page, through the same
// proxy as the app. The tests use it to make test data quickly.
export function api(request: APIRequestContext) {
  const send = async (method: string, path: string, data?: unknown) => {
    const res = await request.fetch(`/playbook-api/api/v1${path}`, { method, data });
    expect(res.ok(), `${method} ${path}: ${await res.text()}`).toBeTruthy();
    return res.status() === 204 ? undefined : res.json();
  };
  return {
    get: (path: string) => send("GET", path),
    post: (path: string, data?: unknown) => send("POST", path, data),
    put: (path: string, data?: unknown) => send("PUT", path, data),
    patch: (path: string, data?: unknown) => send("PATCH", path, data),
  };
}

// newPlaybook makes a playbook that runs one Python step in the sandbox.
export async function newPlaybook(request: APIRequestContext, name = unique("playbook"), code = "print('hello')") {
  const { id } = await api(request).post("/playbooks", { name });
  // The trigger route refuses a playbook with no trigger type.
  await api(request).put(`/playbooks/${id}`, { trigger_type: "manual" });
  await api(request).put(`/playbooks/tasks/${id}`, {
    nodes: [
      { name: "start", operation: "start", parameters: {} },
      {
        name: "hello",
        operation: "python_code",
        connector_id: "code_snippet_py",
        connector_name: "Code Snippet",
        parameters: { code },
      },
    ],
    edges: { start: ["hello"] },
  });
  return { id: id as string, name };
}

export async function newIncident(request: APIRequestContext, title = unique("incident")) {
  const { id } = await api(request).post("/incidents", { title, severity: "medium" });
  return { id: id as string, title };
}

export async function newAlert(request: APIRequestContext, title = unique("alert")) {
  const { id } = await api(request).post("/alerts", { title, severity: "high", source_kind: "edr" });
  return { id: id as string, title };
}

// newUser makes a user with the named builtin roles.
export async function newUser(request: APIRequestContext, roles: string[]) {
  const all: { id: string; name: string }[] = await api(request).get("/roles");
  const username = unique("user");
  const password = "password123";
  await api(request).post("/users", {
    username,
    email: `${username}@ytsoar.test`,
    password,
    role_ids: all.filter((r) => roles.includes(r.name)).map((r) => r.id),
  });
  return { username, password };
}
