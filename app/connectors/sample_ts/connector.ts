// Sample TypeScript connector — copy this folder to start a new TS connector.
// Node runs the file directly by stripping the types (>= 23.6); only erasable
// TS syntax is allowed (no enums, namespaces or parameter properties).
//
//   - require the core with an explicit .ts extension (Node's extensionless
//     require never resolves .ts files)
//   - export a class extending Connector
//   - execute(configs, params, operation) receives the parsed TOML config,
//     the already-templated params, and the operation name from info.json
"use strict";

const { Connector } = require("../core/connector.ts");

type Configs = Record<string, unknown>;
type Params = Record<string, unknown>;

class SampleTS extends Connector {
  async execute(configs: Configs, params: Params, operation: string): Promise<unknown> {
    console.log(`executed, ${operation}`, params); // goes to stderr, never corrupts the result
    return { sample: `executed, ${operation} ${JSON.stringify(params)}` };
  }

  async healthCheck(configs: Configs, params: Params, operation: string): Promise<unknown> {
    return operation;
  }
}

module.exports = { SampleTS };
