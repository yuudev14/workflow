package connectors

import (
	"context"

	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . ConnectorService

type ConnectorService interface {
	GetConnectors(ctx context.Context) ([]domain.ConnectorInfo, error)
	GetConnector(ctx context.Context, connectorID string) (domain.ConnectorInfo, error)
}

type ConnectorServiceImpl struct {
	logger logger.Logger
	store  ConnectorStore
}

func NewConnectorService(log logger.Logger, store ConnectorStore) *ConnectorServiceImpl {
	return &ConnectorServiceImpl{
		logger: log,
		store:  store,
	}
}

func (s *ConnectorServiceImpl) GetConnectors(ctx context.Context) ([]domain.ConnectorInfo, error) {
	return s.store.List(ctx)
}

func (s *ConnectorServiceImpl) GetConnector(ctx context.Context, connectorID string) (domain.ConnectorInfo, error) {
	return s.store.Get(ctx, connectorID)
}
