package connectorstore_test

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"context"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/connectorstore"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func zipReader(t *testing.T, files map[string]string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return r
}

func TestFSWriterExtractAndReplace(t *testing.T) {
	dir := t.TempDir()
	writer := connectorstore.NewFSWriter(logger.NewNop(), dir)

	archive := zipReader(t, map[string]string{
		"my_conn/info.json":            `{"id":"my_conn"}`,
		"my_conn/connector.ts":         "// v1",
		"my_conn/configs/default.toml": `key = "v"`,
	})
	require.NoError(t, writer.Extract(context.Background(), "my_conn", archive, "my_conn/"))

	content, err := os.ReadFile(filepath.Join(dir, "my_conn", "connector.ts"))
	require.NoError(t, err)
	assert.Equal(t, "// v1", string(content))
	assert.FileExists(t, filepath.Join(dir, "my_conn", "configs", "default.toml"))

	// re-upload replaces the folder wholesale: old files must be gone
	replacement := zipReader(t, map[string]string{
		"info.json":    `{"id":"my_conn"}`,
		"connector.py": "# v2",
	})
	require.NoError(t, writer.Extract(context.Background(), "my_conn", replacement, ""))
	assert.FileExists(t, filepath.Join(dir, "my_conn", "connector.py"))
	assert.NoFileExists(t, filepath.Join(dir, "my_conn", "connector.ts"))

	// no staging leftovers
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestFSWriterRemove(t *testing.T) {
	dir := t.TempDir()
	writer := connectorstore.NewFSWriter(logger.NewNop(), dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "my_conn"), 0o755))

	require.NoError(t, writer.Remove(context.Background(), "my_conn"))

	assert.NoDirExists(t, filepath.Join(dir, "my_conn"))
	assert.Error(t, writer.Remove(context.Background(), "../escape"))
}

// The size caps upstream trust the zip's declared UncompressedSize64, so
// extraction must reject an entry whose stream expands past what it declared.
func TestFSWriterRejectsLyingUncompressedSize(t *testing.T) {
	payload := bytes.Repeat([]byte("A"), 1<<20) // 1 MiB that deflates tiny
	var deflated bytes.Buffer
	fw, err := flate.NewWriter(&deflated, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = fw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, fw.Close())

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateRaw(&zip.FileHeader{
		Name:               "my_conn/connector.py",
		Method:             zip.Deflate,
		CRC32:              crc32.ChecksumIEEE(payload),
		CompressedSize64:   uint64(deflated.Len()),
		UncompressedSize64: 10, // lies about the 1 MiB it expands to
	})
	require.NoError(t, err)
	_, err = w.Write(deflated.Bytes())
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	archive, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	dir := t.TempDir()
	writer := connectorstore.NewFSWriter(logger.NewNop(), dir)
	err = writer.Extract(context.Background(), "my_conn", archive, "my_conn/")

	assert.Error(t, err)
	assert.NoDirExists(t, filepath.Join(dir, "my_conn"))
}

func TestFSWriterRejectsUnsafeID(t *testing.T) {
	writer := connectorstore.NewFSWriter(logger.NewNop(), t.TempDir())

	err := writer.Extract(context.Background(), "../escape", zipReader(t, map[string]string{"a": "b"}), "")

	assert.Error(t, err)
}
