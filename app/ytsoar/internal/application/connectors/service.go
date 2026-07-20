package connectors

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . ConnectorService

// MaxUploadBytes caps the uploaded zip; maxUncompressedBytes caps what it may
// expand to (zip-bomb guard).
const (
	MaxUploadBytes       = 50 << 20
	maxUncompressedBytes = 200 << 20
)

// reservedIDs cannot be uploaded: "core" is the shared base-class package,
// the code_snippet pair are virtual connectors implemented by the sandbox,
// and condition/http_request are Go builtins compiled into the worker — an
// upload could never shadow them (the resolver routes builtins first), so
// reject it instead of confusing anyone.
var reservedIDs = map[string]bool{
	"core":            true,
	"code_snippet_py": true,
	"code_snippet_js": true,
	"condition":       true,
	"http_request":    true,
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type ConnectorService interface {
	GetConnectors(ctx context.Context) ([]domain.ConnectorInfo, error)
	GetConnector(ctx context.Context, connectorID string) (domain.ConnectorInfo, error)
	UploadConnector(ctx context.Context, zipBytes []byte, uploadedBy string) (domain.ConnectorInfo, error)
	DeleteConnector(ctx context.Context, connectorID string) error
}

type ConnectorServiceImpl struct {
	logger    logger.Logger
	store     ConnectorStore
	writer    ConnectorWriter
	repo      ConnectorRepository
	installer DepsInstaller
}

func NewConnectorService(log logger.Logger, store ConnectorStore, writer ConnectorWriter,
	repo ConnectorRepository, installer DepsInstaller) *ConnectorServiceImpl {
	return &ConnectorServiceImpl{
		logger:    log,
		store:     store,
		writer:    writer,
		repo:      repo,
		installer: installer,
	}
}

func (s *ConnectorServiceImpl) GetConnectors(ctx context.Context) ([]domain.ConnectorInfo, error) {
	return s.store.List(ctx)
}

func (s *ConnectorServiceImpl) GetConnector(ctx context.Context, connectorID string) (domain.ConnectorInfo, error) {
	return s.store.Get(ctx, connectorID)
}

// UploadConnector validates the zip, extracts it into the tree (replacing any
// previous version), vendors its declared dependencies and upserts the audit
// row. Validation failures wrap ErrInvalidConnector.
func (s *ConnectorServiceImpl) UploadConnector(ctx context.Context, zipBytes []byte, uploadedBy string) (domain.ConnectorInfo, error) {
	if len(zipBytes) > MaxUploadBytes {
		return nil, apperr.Wrap(apperr.Invalid, fmt.Sprintf("zip exceeds %d bytes", MaxUploadBytes), ErrInvalidConnector)
	}
	archive, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, apperr.Wrap(apperr.Invalid, fmt.Sprintf("not a zip archive: %v", err), ErrInvalidConnector)
	}

	prefix, err := validateArchive(archive)
	if err != nil {
		return nil, err
	}
	meta, err := readArchiveInfo(archive, prefix)
	if err != nil {
		return nil, err
	}

	if err := s.writer.Extract(ctx, meta.id, archive, prefix); err != nil {
		return nil, err
	}
	if err := s.installer.Install(ctx, meta.id); err != nil {
		s.removeQuietly(ctx, meta.id)
		return nil, fmt.Errorf("dependency install failed for %s: %w", meta.id, err)
	}

	sum := sha256.Sum256(zipBytes)
	if _, err := s.repo.Upsert(ctx, domain.ConnectorRecord{
		ID:         meta.id,
		Name:       meta.name,
		Runtime:    meta.runtime,
		Version:    meta.version,
		Checksum:   hex.EncodeToString(sum[:]),
		UploadedBy: uploadedBy,
	}); err != nil {
		s.removeQuietly(ctx, meta.id)
		return nil, err
	}

	s.logger.Infow("connector uploaded",
		"connector", meta.id, "runtime", meta.runtime, "uploaded_by", uploadedBy)
	return s.store.Get(ctx, meta.id)
}

func (s *ConnectorServiceImpl) DeleteConnector(ctx context.Context, connectorID string) error {
	// reservedIDs guard deletes too: removing a builtin's info.json dir would
	// strip it from the editor even though its implementation is compiled in.
	if reservedIDs[connectorID] {
		return apperr.Wrap(apperr.Invalid, fmt.Sprintf("connector id %q is reserved", connectorID), ErrInvalidConnector)
	}
	if _, err := s.store.Get(ctx, connectorID); err != nil {
		return err
	}
	if err := s.writer.Remove(ctx, connectorID); err != nil {
		return err
	}
	// built-ins seeded from the repo have no audit row — not an error
	if err := s.repo.Delete(ctx, connectorID); err != nil && err != ErrConnectorNotFound {
		return err
	}
	s.logger.Infow("connector deleted", "connector", connectorID)
	return nil
}

func (s *ConnectorServiceImpl) removeQuietly(ctx context.Context, connectorID string) {
	if err := s.writer.Remove(ctx, connectorID); err != nil {
		s.logger.Errorw("could not clean up connector after failed upload",
			"connector", connectorID, "error", err)
	}
}

type archiveInfo struct {
	id      string
	name    string
	runtime string
	version string
}

// validateArchive rejects unsafe entries and returns the single top-level
// folder to strip ("" when files sit at the zip root).
func validateArchive(archive *zip.Reader) (string, error) {
	if len(archive.File) == 0 {
		return "", apperr.Wrap(apperr.Invalid, "empty zip", ErrInvalidConnector)
	}
	var total uint64
	prefix := ""
	prefixSet := false
	for _, file := range archive.File {
		name := file.Name
		if strings.Contains(name, `\`) || strings.HasPrefix(name, "/") {
			return "", apperr.Wrap(apperr.Invalid, fmt.Sprintf("unsafe path %q", name), ErrInvalidConnector)
		}
		clean := path.Clean(name)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return "", apperr.Wrap(apperr.Invalid, fmt.Sprintf("unsafe path %q", name), ErrInvalidConnector)
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return "", apperr.Wrap(apperr.Invalid, fmt.Sprintf("symlink %q not allowed", name), ErrInvalidConnector)
		}
		total += file.UncompressedSize64
		if total > maxUncompressedBytes {
			return "", apperr.Wrap(apperr.Invalid, fmt.Sprintf("archive expands beyond %d bytes", maxUncompressedBytes), ErrInvalidConnector)
		}
		if strings.HasSuffix(name, "/") {
			continue // directory entry
		}
		top, rest, found := strings.Cut(clean, "/")
		entryPrefix := ""
		if found && rest != "" {
			entryPrefix = top + "/"
		}
		if !prefixSet {
			prefix, prefixSet = entryPrefix, true
			continue
		}
		if prefix != "" && !strings.HasPrefix(clean, prefix) {
			prefix = "" // mixed layout -> treat the zip root as the connector root
		}
	}
	return prefix, nil
}

// readArchiveInfo parses info.json at the connector root and checks the
// runtime implementation file is present.
func readArchiveInfo(archive *zip.Reader, prefix string) (archiveInfo, error) {
	files := map[string]bool{}
	var raw []byte
	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		rel := strings.TrimPrefix(path.Clean(file.Name), prefix)
		files[rel] = true
		if rel != "info.json" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return archiveInfo{}, err
		}
		raw, err = io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return archiveInfo{}, err
		}
	}
	if raw == nil {
		return archiveInfo{}, apperr.Wrap(apperr.Invalid, "info.json missing at the connector root", ErrInvalidConnector)
	}

	var info struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Runtime string `json:"runtime"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return archiveInfo{}, apperr.Wrap(apperr.Invalid, fmt.Sprintf("invalid info.json: %v", err), ErrInvalidConnector)
	}
	if !idPattern.MatchString(info.ID) {
		return archiveInfo{}, apperr.Wrap(apperr.Invalid, fmt.Sprintf("invalid connector id %q", info.ID), ErrInvalidConnector)
	}
	if reservedIDs[info.ID] {
		return archiveInfo{}, apperr.Wrap(apperr.Invalid, fmt.Sprintf("connector id %q is reserved", info.ID), ErrInvalidConnector)
	}
	if info.Name == "" {
		return archiveInfo{}, apperr.Wrap(apperr.Invalid, "info.json needs a name", ErrInvalidConnector)
	}
	if info.Runtime == "" {
		info.Runtime = "python"
	}

	switch info.Runtime {
	case "node":
		if !files["connector.ts"] && !files["connector.js"] {
			return archiveInfo{}, apperr.Wrap(apperr.Invalid, "runtime node needs connector.ts or connector.js", ErrInvalidConnector)
		}
	case "python":
		if !files["connector.py"] {
			return archiveInfo{}, apperr.Wrap(apperr.Invalid, "runtime python needs connector.py", ErrInvalidConnector)
		}
	default:
		return archiveInfo{}, apperr.Wrap(apperr.Invalid, fmt.Sprintf("unknown runtime %q", info.Runtime), ErrInvalidConnector)
	}

	return archiveInfo{id: info.ID, name: info.Name, runtime: info.Runtime, version: info.Version}, nil
}
