// Sample JavaScript connector — the JS mirror of the Python contract.
// Export a class extending core/connector.js's Connector; execute() receives
// the parsed TOML config, the already-templated params, and the operation
// name, and dispatches however it likes.
"use strict";

const { Connector } = require("../core/connector");

class HttpRequestConnector extends Connector {
  async execute(configs, params, operation) {
    const operations = {
      get_request: (cfg, p) => this.getRequest(cfg, p),
    };
    const handler = operations[operation];
    if (!handler) {
      throw new Error(
        `operation (${operation}) does not exist in ${this.constructor.name}`
      );
    }
    return handler(configs, params);
  }

  async healthCheck() {
    return true;
  }

  async getRequest(configs, params) {
    const response = await fetch(params.url, {
      headers: (configs && configs.headers) || {},
    });
    const text = await response.text();
    let body;
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
    return { status: response.status, body };
  }
}

module.exports = { HttpRequestConnector };
