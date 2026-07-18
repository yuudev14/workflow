// Runs inside a fresh `node --input-type=commonjs-typescript` child spawned
// by the sandbox — Node strips the types natively, no build step.
// Payload arrives as JSON on stdin, the result leaves as JSON on stdout.
// Mirrors the Python code node: templates render over params (including the
// code string itself), the snippet runs with `params`/`steps` in scope and
// sets `result`, and the output is {"code_output": result}.
//
// NOTE: the user snippet itself stays JavaScript — it runs through
// `new AsyncFunction(...)`, which type stripping does not touch.
"use strict";

interface CodePayload {
  params?: Record<string, unknown>;
  steps?: Record<string, unknown>;
}

function render(value: unknown, variables: Record<string, unknown>): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => render(item, variables));
  }
  if (value && typeof value === "object") {
    const out: Record<string, unknown> = {};
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

const chunks: Buffer[] = [];
process.stdin.on("data", (chunk: Buffer) => chunks.push(chunk));
process.stdin.on("end", async () => {
  try {
    const payload: CodePayload = JSON.parse(Buffer.concat(chunks).toString());
    const variables = { steps: payload.steps || {} };
    const params = render(payload.params || {}, variables) as Record<string, unknown>;
    const code = typeof params.code === "string" ? params.code : "";

    // stdout is the JSON result channel: swap in a console bound entirely to
    // stderr so no console method (log, dir, table, group, ...) can corrupt it.
    const writeResult = process.stdout.write.bind(process.stdout);
    globalThis.console = new (require("node:console").Console)(
      process.stderr,
      process.stderr
    );

    const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
    const fn = new AsyncFunction(
      "params",
      "steps",
      `${code}\n;return typeof result === "undefined" ? undefined : result;`
    );
    const result: unknown = await fn(params, variables.steps);
    // Exit once stdout has flushed. Without this, any snippet that leaves the
    // event loop alive (a timer, an open socket, a fetch keep-alive pool) would
    // keep the process running until the node timeout SIGKILLs it.
    writeResult(
      JSON.stringify({ code_output: result === undefined ? null : result }),
      () => process.exit(0)
    );
  } catch (err) {
    console.error(err instanceof Error && err.stack ? err.stack : String(err));
    process.exit(1);
  }
});
