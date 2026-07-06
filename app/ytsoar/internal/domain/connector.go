package domain

// ConnectorInfo is a connector's parsed info.json plus the derived "configs"
// list (config file stems). It stays a map so unknown metadata fields survive
// the trip to the frontend unchanged — shape parity with the old FastAPI
// GET /connectors/ endpoint.
type ConnectorInfo map[string]any
