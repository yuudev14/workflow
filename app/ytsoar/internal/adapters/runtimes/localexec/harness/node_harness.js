// Runs inside a fresh `node` child spawned by the Go worker.
// Payload arrives as JSON on stdin, the result leaves as JSON on stdout.
// Mirrors the Python code node: templates render over params (including the
// code string itself), the snippet runs with `params`/`steps` in scope and
// sets `result`, and the output is {"code_output": result}.
"use strict";

function render(value, variables) {
  if (Array.isArray(value)) {
    return value.map((item) => render(item, variables));
  }
  if (value && typeof value === "object") {
    const out = {};
    for (const [key, val] of Object.entries(value)) {
      out[key] = render(val, variables);
    }
    return out;
  }
  if (typeof value === "string" && (value.includes("{{") || value.includes("{%"))) {
    // lazy require: plain snippets run even without nunjucks on NODE_PATH
    const nunjucks = require("nunjucks");
    return nunjucks.renderString(value, { var: variables });
  }
  return value;
}

const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(chunk));
process.stdin.on("end", async () => {
  try {
    const payload = JSON.parse(Buffer.concat(chunks).toString());
    const variables = { steps: payload.steps || {} };
    const params = render(payload.params || {}, variables);
    const code = params.code || "";

    const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
    const fn = new AsyncFunction(
      "params",
      "steps",
      `${code}\n;return typeof result === "undefined" ? undefined : result;`
    );
    const result = await fn(params, variables.steps);
    process.stdout.write(
      JSON.stringify({ code_output: result === undefined ? null : result })
    );
  } catch (err) {
    console.error(err && err.stack ? err.stack : String(err));
    process.exit(1);
  }
});
