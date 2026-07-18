// Compatibility shim: the implementation lives in connector.ts (Node strips
// the types natively). Plain-JS connectors keep working because their
// extensionless require("../core/connector") resolves this file — Node never
// resolves .ts without an explicit extension. Requiring either path yields
// the SAME module instance, so `instanceof Connector` holds across JS and TS
// connectors.
"use strict";

module.exports = require("./connector.ts");
