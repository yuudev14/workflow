// Runs inside a fresh `node` child spawned by the Go worker to execute one
// JS connector operation. Payload: { connectors_dir, connector_id, operation,
// config, params, steps } — the TOML config is already parsed by Go.
// Connector contract: connectors-node/<id>/connector.js exports
// `operations = { opName(config, params) { ... } }`.
"use strict";

const path = require("node:path");

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

    const connectorPath = path.join(
      payload.connectors_dir,
      payload.connector_id,
      "connector.js"
    );
    const connector = require(connectorPath);
    const operation = connector.operations && connector.operations[payload.operation];
    if (typeof operation !== "function") {
      throw new Error(
        `operation (${payload.operation}) does not exist in connector ${payload.connector_id}`
      );
    }

    const result = await operation(payload.config || {}, params);
    process.stdout.write(JSON.stringify(result === undefined ? null : result));
  } catch (err) {
    console.error(err && err.stack ? err.stack : String(err));
    process.exit(1);
  }
});
