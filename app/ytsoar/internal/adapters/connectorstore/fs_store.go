// Package connectorstore reads connector metadata from the unified
// connectors tree: <dir>/<id>/{info.json, configs/*.toml}. It implements
// connectors.ConnectorStore with the same semantics as the old FastAPI
// GET /connectors/ endpoint (skip "core", skip broken entries, append the
// config stems as "configs").
package connectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type FSStore struct {
	logger logger.Logger
	dir    string
}

func NewFSStore(log logger.Logger, dir string) *FSStore {
	return &FSStore{
		logger: log,
		dir:    dir,
	}
}

func (s *FSStore) List(ctx context.Context) ([]domain.ConnectorInfo, error) {
	infos := []domain.ConnectorInfo{}
	if s.dir == "" {
		return infos, nil
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "core" {
			continue
		}
		info, err := s.readInfo(entry.Name())
		if err != nil {
			// parity with FastAPI: a broken connector never breaks the list
			s.logger.Warnw("skipping connector with unreadable info.json",
				"connector", entry.Name(), "error", err)
			continue
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (s *FSStore) Get(ctx context.Context, connectorID string) (domain.ConnectorInfo, error) {
	if s.dir == "" || !isSafeID(connectorID) {
		return nil, connectors.ErrConnectorNotFound
	}
	info, err := s.readInfo(connectorID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connectors.ErrConnectorNotFound
		}
		return nil, err
	}
	return info, nil
}

func (s *FSStore) readInfo(connectorID string) (domain.ConnectorInfo, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, connectorID, "info.json"))
	if err != nil {
		return nil, err
	}
	info := domain.ConnectorInfo{}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("invalid info.json: %w", err)
	}

	configsDir := filepath.Join(s.dir, connectorID, "configs")
	if entries, err := os.ReadDir(configsDir); err == nil {
		configs := []string{}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			configs = append(configs, strings.TrimSuffix(name, filepath.Ext(name)))
		}
		info["configs"] = configs
	}
	return info, nil
}

// isSafeID rejects ids that could escape the tree (path traversal).
func isSafeID(id string) bool {
	return id != "" && id != "." && id != ".." &&
		!strings.ContainsAny(id, `/\`)
}
