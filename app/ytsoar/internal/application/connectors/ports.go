package connectors

import (
	"context"
	"errors"

	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/store_mock.go -package=mocks . ConnectorStore

// ErrConnectorNotFound is returned when the requested connector id does not
// exist in the tree.
var ErrConnectorNotFound = errors.New("connector not found")

// ConnectorStore reads connector metadata from the unified connectors tree.
// The API process is the tree's only writer (uploads land here in a later
// phase); the sandbox only executes from it.
type ConnectorStore interface {
	List(ctx context.Context) ([]domain.ConnectorInfo, error)
	Get(ctx context.Context, connectorID string) (domain.ConnectorInfo, error)
}
