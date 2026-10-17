package domain

import "time"

// ConnectorInfo is the parsed info.json of a connector plus the derived "configs" list
// (config file stems). It stays a map, so unknown metadata fields reach the frontend
// unchanged
type ConnectorInfo map[string]any

// ConnectorRecord is the audit and control row in Postgres for each uploaded connector.
// The filesystem tree is still the source of truth for execution. This row keeps the
// provenance (zip checksum, uploader) and the disable flag.
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
