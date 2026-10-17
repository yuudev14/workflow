package connectors

import (
	"archive/zip"
	"context"

	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

//go:generate mockgen -destination=mocks/store_mock.go -package=mocks . ConnectorStore,ConnectorWriter,ConnectorRepository,DepsInstaller

// ErrConnectorNotFound: the connector id is not in the tree.
var ErrConnectorNotFound = apperr.New(apperr.NotFound, "connector not found")

// ErrInvalidConnector: upload validation failed (bad zip, missing info.json, unsafe
// paths, ...). Handlers return 400.
var ErrInvalidConnector = apperr.New(apperr.Invalid, "invalid connector")

// ConnectorStore reads connector metadata from the unified connectors tree.
type ConnectorStore interface {
	List(ctx context.Context) ([]domain.ConnectorInfo, error)
	Get(ctx context.Context, connectorID string) (domain.ConnectorInfo, error)
}

// ConnectorWriter changes the tree. Only the API process writes it, the sandbox mounts it
// read-only.
type ConnectorWriter interface {
	// Extract writes the archive to <tree>/<connectorID> and replaces the old folder. prefix
	// is the one top-level zip folder to strip ("" when files are at the zip root). The
	// entries were validated before.
	Extract(ctx context.Context, connectorID string, archive *zip.Reader, prefix string) error
	Remove(ctx context.Context, connectorID string) error
}

// ConnectorRepository persists the audit rows in Postgres.
type ConnectorRepository interface {
	Upsert(ctx context.Context, record domain.ConnectorRecord) (domain.ConnectorRecord, error)
	// Delete returns ErrConnectorNotFound when no row matched.
	Delete(ctx context.Context, connectorID string) error
}

// DepsInstaller vendors the declared dependencies of a connector (requirements.txt ->
// deps/, package.json -> node_modules/). It is the step of `make connector-deps`, run at
// upload.
type DepsInstaller interface {
	Install(ctx context.Context, connectorID string) error
}
