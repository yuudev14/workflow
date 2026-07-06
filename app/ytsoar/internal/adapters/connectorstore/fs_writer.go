package connectorstore

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuudev14/ytsoar/internal/logger"
)

// FSWriter mutates the unified connectors tree. It implements
// connectors.ConnectorWriter; the API process is the tree's only writer.
type FSWriter struct {
	logger logger.Logger
	dir    string
}

func NewFSWriter(log logger.Logger, dir string) *FSWriter {
	return &FSWriter{
		logger: log,
		dir:    dir,
	}
}

// Extract unpacks the (already validated) archive into <dir>/<connectorID>.
// It stages into a temp folder first so a failed extraction never leaves a
// half-written connector, then swaps it in place of any previous version.
func (w *FSWriter) Extract(ctx context.Context, connectorID string, archive *zip.Reader, prefix string) error {
	if w.dir == "" {
		return fmt.Errorf("connector uploads are disabled: CONNECTORS_DIR is not set")
	}
	if !isSafeID(connectorID) {
		return fmt.Errorf("unsafe connector id %q", connectorID)
	}

	staging, err := os.MkdirTemp(w.dir, ".upload-"+connectorID+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/") {
			continue
		}
		rel := strings.TrimPrefix(path.Clean(file.Name), prefix)
		dest := filepath.Join(staging, filepath.FromSlash(rel))
		if !strings.HasPrefix(dest, staging+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe archive path %q", file.Name)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := writeArchiveFile(file, dest); err != nil {
			return err
		}
	}

	target := filepath.Join(w.dir, connectorID)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.Rename(staging, target); err != nil {
		return err
	}
	w.logger.Debugw("connector extracted", "connector", connectorID, "dir", target)
	return nil
}

func (w *FSWriter) Remove(ctx context.Context, connectorID string) error {
	if w.dir == "" || !isSafeID(connectorID) {
		return fmt.Errorf("unsafe connector id %q", connectorID)
	}
	return os.RemoveAll(filepath.Join(w.dir, connectorID))
}

func writeArchiveFile(file *zip.File, dest string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, reader)
	return err
}
