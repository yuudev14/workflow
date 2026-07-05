// Example JS connector. Contract: export `operations`, an object mapping
// operation names to async (config, params) => result functions. Params are
// already jinja-rendered by the harness; config comes from
// configs/<name>.toml parsed by the Go worker.
"use strict";

module.exports.operations = {
  async get_request(config, params) {
    const response = await fetch(params.url, {
      headers: (config && config.headers) || {},
    });
    const text = await response.text();
    let body;
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
    return { status: response.status, body };
  },
};
