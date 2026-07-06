// Runs inside a fresh `node --input-type=commonjs-typescript` child spawned
// by the sandbox to execute one JS/TS connector operation — Node strips the
// types natively, no build step. Payload: { connectors_dir, connector_id,
// operation, config, params, steps } — the TOML config is already parsed by Go.
//
// Mirror of the python connector harness: the tree's own core
// (<connectors_dir>/core/connector.ts) provides the Connector base class,
// class discovery (connector.ts OR connector.js entry) and templating, so
// connector semantics live with the connectors, not in this binary.
"use strict";

const path = require("node:path");

interface ConnectorPayload {
  connectors_dir: string;
  connector_id: string;
  operation: string;
  config?: Record<string, unknown>;
  params?: Record<string, unknown>;
  steps?: Record<string, unknown>;
}

const chunks: Buffer[] = [];
process.stdin.on("data", (chunk: Buffer) => chunks.push(chunk));
process.stdin.on("end", async () => {
  try {
    const payload: ConnectorPayload = JSON.parse(Buffer.concat(chunks).toString());
    const core = require(path.join(payload.connectors_dir, "core", "connector.ts"));

    // stdout is the JSON result channel: reroute connector console output to
    // stderr so it cannot corrupt it.
    const writeResult = process.stdout.write.bind(process.stdout);
    for (const level of ["log", "info", "warn", "debug"] as const) {
      console[level] = (...args: unknown[]) => console.error(...args);
    }

    const connector = core.getClassContainer(
      payload.connectors_dir,
      payload.connector_id
    );
    const params = core.evaluateParams(payload.params || {}, {
      steps: payload.steps || {},
    });
    const result: unknown = await connector.execute(
      payload.config || {},
      params,
      payload.operation
    );
    writeResult(JSON.stringify(result === undefined ? null : result));
  } catch (err) {
    console.error(err instanceof Error && err.stack ? err.stack : String(err));
    process.exit(1);
  }
});
