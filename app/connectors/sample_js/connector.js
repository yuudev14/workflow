// Sample JavaScript connector — copy this folder to start a new JS connector.
// The JS mirror of connectors/sample/connector.py:
//   - export a class extending core/connector.js's Connector
//   - execute(configs, params, operation) receives the parsed TOML config,
//     the already-templated params, and the operation name from info.json
"use strict";

const { Connector } = require("../core/connector");

class SampleJS extends Connector {
  async execute(configs, params, operation) {
    console.log(`executed, ${operation}`, params); // goes to stderr, never corrupts the result
    return { sample: `executed, ${operation} ${JSON.stringify(params)}` };
  }

  async healthCheck(configs, params, operation) {
    return operation;
  }
}

module.exports = { SampleJS };
