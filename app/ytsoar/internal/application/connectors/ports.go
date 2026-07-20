package connectors

import (
	"archive/zip"
	"context"

	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

//go:generate mockgen -destination=mocks/store_mock.go -package=mocks . ConnectorStore,ConnectorWriter,ConnectorRepository,DepsInstaller

// ErrConnectorNotFound is returned when the requested connector id does not
// exist in the tree.
var ErrConnectorNotFound = apperr.New(apperr.NotFound, "connector not found")

// ErrInvalidConnector marks upload validation failures (bad zip, missing
// info.json, unsafe paths, ...) — handlers map it to 400.
var ErrInvalidConnector = apperr.New(apperr.Invalid, "invalid connector")

// ConnectorStore reads connector metadata from the unified connectors tree.
type ConnectorStore interface {
	List(ctx context.Context) ([]domain.ConnectorInfo, error)
	Get(ctx context.Context, connectorID string) (domain.ConnectorInfo, error)
}

// ConnectorWriter mutates the tree. The API process is the tree's only
// writer; the sandbox mounts it read-only.
type ConnectorWriter interface {
	// Extract writes the archive under <tree>/<connectorID>, replacing any
	// existing folder. prefix is the single top-level zip folder to strip
	// ("" when files sit at the zip root). Entries were validated upstream.
	Extract(ctx context.Context, connectorID string, archive *zip.Reader, prefix string) error
	Remove(ctx context.Context, connectorID string) error
}

// ConnectorRepository persists the audit rows in Postgres.
type ConnectorRepository interface {
	Upsert(ctx context.Context, record domain.ConnectorRecord) (domain.ConnectorRecord, error)
	// Delete returns ErrConnectorNotFound when no row matched.
	Delete(ctx context.Context, connectorID string) error
}

// DepsInstaller vendors a connector's declared dependencies
// (requirements.txt -> deps/, package.json -> node_modules/) — the same step
// `make connector-deps` runs, executed at upload time.
type DepsInstaller interface {
	Install(ctx context.Context, connectorID string) error
}
