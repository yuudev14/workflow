package domain

import "time"

// ConnectorInfo is a connector's parsed info.json plus the derived "configs"
// list (config file stems). It stays a map so unknown metadata fields survive
// the trip to the frontend unchanged — shape parity with the old FastAPI
// GET /connectors/ endpoint.
type ConnectorInfo map[string]any

// ConnectorRecord is the audit/control row kept in Postgres for every
// uploaded connector. The filesystem tree stays the execution source of
// truth; this records provenance (zip checksum, uploader) and the disable
// flag.
type ConnectorRecord struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Runtime    string    `json:"runtime"`
	Version    string    `json:"version"`
	Checksum   string    `json:"checksum"`
	UploadedBy string    `json:"uploaded_by"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
