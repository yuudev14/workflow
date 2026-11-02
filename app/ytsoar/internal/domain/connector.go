package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ConnectorInfo is the parsed info.json of a connector. It stays a map, so unknown metadata
// fields reach the frontend unchanged
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

// ConnectorConfig is one named config of a connector, for example "prod". A step picks it
// by name, or gets the default config when it names none.
type ConnectorConfig struct {
	ID          uuid.UUID       `json:"id"`
	ConnectorID string          `json:"connector_id"`
	Name        string          `json:"name"`
	Config      json.RawMessage `json:"config"`
	IsDefault   bool            `json:"is_default"`
	CreatedBy   *uuid.UUID      `json:"created_by"`
	UpdatedBy   *uuid.UUID      `json:"updated_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
