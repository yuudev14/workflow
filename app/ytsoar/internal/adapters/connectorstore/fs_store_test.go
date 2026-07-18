package connectorstore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/connectorstore"
	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func sampleTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	write("sample/info.json", `{"id":"sample","name":"Sample","runtime":"python","operations":[]}`)
	write("sample/configs/default.toml", `key = "value"`)
	write("sample/configs/prod.toml", `key = "prod"`)
	write("http_js/info.json", `{"id":"http_js","name":"HTTP JS","runtime":"node"}`)
	write("broken/info.json", `{not json`)
	write("core/connector.py", "# core is never listed")
	return dir
}

func TestFSStoreList(t *testing.T) {
	store := connectorstore.NewFSStore(logger.NewNop(), sampleTree(t))

	infos, err := store.List(context.Background())

	require.NoError(t, err)
	require.Len(t, infos, 2) // broken skipped, core skipped
	byID := map[string]bool{}
	for _, info := range infos {
		byID[info["id"].(string)] = true
	}
	assert.True(t, byID["sample"])
	assert.True(t, byID["http_js"])
}

func TestFSStoreGetAddsConfigStems(t *testing.T) {
	store := connectorstore.NewFSStore(logger.NewNop(), sampleTree(t))

	info, err := store.Get(context.Background(), "sample")

	require.NoError(t, err)
	assert.Equal(t, "Sample", info["name"])
	assert.ElementsMatch(t, []string{"default", "prod"}, info["configs"])
}

func TestFSStoreGetNotFound(t *testing.T) {
	store := connectorstore.NewFSStore(logger.NewNop(), sampleTree(t))

	_, err := store.Get(context.Background(), "ghost")

	assert.ErrorIs(t, err, connectors.ErrConnectorNotFound)
}

func TestFSStoreGetRejectsPathTraversal(t *testing.T) {
	store := connectorstore.NewFSStore(logger.NewNop(), sampleTree(t))

	for _, id := range []string{"..", "../secrets", "a/b", `a\b`, ""} {
		_, err := store.Get(context.Background(), id)
		assert.ErrorIs(t, err, connectors.ErrConnectorNotFound, "id=%q", id)
	}
}

func TestFSStoreEmptyDirListsNothing(t *testing.T) {
	store := connectorstore.NewFSStore(logger.NewNop(), "")

	infos, err := store.List(context.Background())

	require.NoError(t, err)
	assert.Empty(t, infos)
}
