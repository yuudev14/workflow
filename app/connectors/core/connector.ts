// Base class + shared helpers for JavaScript/TypeScript connectors — the
// mirror of core/connector.py. Node strips the types natively (>= 23.6), so
// this file runs as-is; only erasable TS syntax is allowed (no enums,
// namespaces or parameter properties).
//
// Connector authors write (TS shown; JS is the same without the types and
// requires "../core/connector" instead):
//
//   const { Connector } = require("../core/connector.ts");
//
//   class MyConnector extends Connector {
//     async execute(configs: Configs, params: Params, operation: string) { ... }
//     async healthCheck(configs: Configs, params: Params, operation: string) { ... }
//   }
//
//   module.exports = { MyConnector };
//
// configs arrive already parsed (the sandbox parses configs/<name>.toml);
// params are rendered by evaluateParams before execute is called.
"use strict";

const fs = require("node:fs");
const path = require("node:path");

type Configs = Record<string, unknown>;
type Params = Record<string, unknown>;

class Connector {
  // eslint-disable-next-line no-unused-vars
  async execute(
    configs: Configs,
    params: Params,
    operation: string,
    ...args: unknown[]
  ): Promise<unknown> {
    throw new Error(
      `execute function is not implemented in ${this.constructor.name}`
    );
  }

  // eslint-disable-next-line no-unused-vars
  async healthCheck(
    configs: Configs,
    params: Params,
    operation: string,
    ...args: unknown[]
  ): Promise<unknown> {
    throw new Error(
      `healthCheck function is not implemented in ${this.constructor.name}`
    );
  }
}

type ConnectorClass = new () => Connector;

// resolveEntry picks the connector's implementation file: TypeScript first,
// then JavaScript. Explicit extensions are required because Node's
// extensionless require never resolves .ts files.
function resolveEntry(connectorsDir: string, connectorId: string): string {
  for (const entry of ["connector.ts", "connector.js"]) {
    const candidate = path.join(connectorsDir, connectorId, entry);
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  throw new Error(
    `no connector.ts or connector.js found in connector ${connectorId}`
  );
}

// getClassContainer mirrors Connector.get_class_container: load
// <connectorsDir>/<connectorId>/connector.{ts,js} and return an instance of
// the first exported class that extends Connector.
function getClassContainer(
  connectorsDir: string,
  connectorId: string
): Connector {
  const exported: unknown = require(resolveEntry(connectorsDir, connectorId));
  const candidates: unknown[] =
    typeof exported === "function"
      ? [exported]
      : Object.values(exported || {});
  for (const candidate of candidates) {
    if (
      typeof candidate === "function" &&
      candidate.prototype instanceof Connector
    ) {
      return new (candidate as ConnectorClass)();
    }
  }
  throw new Error(
    `no class extending Connector found in connector ${connectorId}`
  );
}

// evaluateParams mirrors Connector.evaluate_params: recursively render
// template strings against the variables using nunjucks, so
// {{ var.steps["node name"] }} works the same as the jinja2 side.
function evaluateParams(
  parameters: unknown,
  variables: Record<string, unknown>
): unknown {
  if (parameters === null || parameters === undefined) {
    return {};
  }
  if (Array.isArray(parameters)) {
    return parameters.map((item) => evaluateParams(item, variables));
  }
  if (typeof parameters === "object") {
    const out: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(parameters)) {
      out[key] = evaluateParams(value, variables);
    }
    return out;
  }
  if (
    typeof parameters === "string" &&
    (parameters.includes("{{") || parameters.includes("{%"))
  ) {
    const nunjucks = require("nunjucks"); // lazy: plain values need no nunjucks install
    return nunjucks.renderString(parameters, { var: variables });
  }
  return parameters;
}

module.exports = { Connector, getClassContainer, evaluateParams };
