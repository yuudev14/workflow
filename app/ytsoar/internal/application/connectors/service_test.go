package connectors_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/application/connectors/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func buildZip(t *testing.T, files map[string]string) []byte {
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
	return buf.Bytes()
}

type serviceMocks struct {
	store     *mocks.MockConnectorStore
	writer    *mocks.MockConnectorWriter
	repo      *mocks.MockConnectorRepository
	installer *mocks.MockDepsInstaller
}

func newService(t *testing.T) (*connectors.ConnectorServiceImpl, serviceMocks) {
	ctrl := gomock.NewController(t)
	m := serviceMocks{
		store:     mocks.NewMockConnectorStore(ctrl),
		writer:    mocks.NewMockConnectorWriter(ctrl),
		repo:      mocks.NewMockConnectorRepository(ctrl),
		installer: mocks.NewMockDepsInstaller(ctrl),
	}
	svc := connectors.NewConnectorService(logger.NewNop(), m.store, m.writer, m.repo, m.installer)
	return svc, m
}

func TestUploadConnectorPython(t *testing.T) {
	svc, m := newService(t)
	zipBytes := buildZip(t, map[string]string{
		"info.json":    `{"id":"my_conn","name":"My Connector","runtime":"python","version":"1.2"}`,
		"connector.py": "class C: pass",
	})

	m.writer.EXPECT().Extract(gomock.Any(), "my_conn", gomock.Any(), "").Return(nil)
	m.installer.EXPECT().Install(gomock.Any(), "my_conn").Return(nil)
	m.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, rec domain.ConnectorRecord) (domain.ConnectorRecord, error) {
			assert.Equal(t, "my_conn", rec.ID)
			assert.Equal(t, "My Connector", rec.Name)
			assert.Equal(t, "python", rec.Runtime)
			assert.Equal(t, "1.2", rec.Version)
			assert.NotEmpty(t, rec.Checksum)
			assert.Equal(t, "alice", rec.UploadedBy)
			return rec, nil
		})
	m.store.EXPECT().Get(gomock.Any(), "my_conn").
		Return(domain.ConnectorInfo{"id": "my_conn"}, nil)

	info, err := svc.UploadConnector(context.Background(), zipBytes, "alice")

	require.NoError(t, err)
	assert.Equal(t, "my_conn", info["id"])
}

func TestUploadConnectorStripsTopLevelFolder(t *testing.T) {
	svc, m := newService(t)
	zipBytes := buildZip(t, map[string]string{
		"my_conn/info.json":            `{"id":"my_conn","name":"My","runtime":"node"}`,
		"my_conn/connector.ts":         "// ts",
		"my_conn/configs/default.toml": `key = "v"`,
	})

	m.writer.EXPECT().Extract(gomock.Any(), "my_conn", gomock.Any(), "my_conn/").Return(nil)
	m.installer.EXPECT().Install(gomock.Any(), "my_conn").Return(nil)
	m.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).
		Return(domain.ConnectorRecord{}, nil)
	m.store.EXPECT().Get(gomock.Any(), "my_conn").
		Return(domain.ConnectorInfo{"id": "my_conn"}, nil)

	_, err := svc.UploadConnector(context.Background(), zipBytes, "")

	require.NoError(t, err)
}

func TestUploadConnectorValidation(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
	}{
		{"missing info.json", map[string]string{"connector.py": "x"}},
		{"path traversal", map[string]string{
			"info.json":    `{"id":"ok_id","name":"n","runtime":"python"}`,
			"../evil.py":   "x",
			"connector.py": "x",
		}},
		{"node without entry file", map[string]string{
			"info.json": `{"id":"ok_id","name":"n","runtime":"node"}`,
		}},
		{"python without connector.py", map[string]string{
			"info.json": `{"id":"ok_id","name":"n","runtime":"python"}`,
		}},
		{"reserved id", map[string]string{
			"info.json":    `{"id":"core","name":"n","runtime":"python"}`,
			"connector.py": "x",
		}},
		{"bad id", map[string]string{
			"info.json":    `{"id":"No Spaces!","name":"n","runtime":"python"}`,
			"connector.py": "x",
		}},
		{"unknown runtime", map[string]string{
			"info.json":    `{"id":"ok_id","name":"n","runtime":"ruby"}`,
			"connector.py": "x",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newService(t) // no mock expectations: nothing may be written
			_, err := svc.UploadConnector(context.Background(), buildZip(t, tc.files), "")
			assert.ErrorIs(t, err, connectors.ErrInvalidConnector)
		})
	}
}

func TestUploadConnectorNotAZip(t *testing.T) {
	svc, _ := newService(t)

	_, err := svc.UploadConnector(context.Background(), []byte("plain text"), "")

	assert.ErrorIs(t, err, connectors.ErrInvalidConnector)
}

func TestUploadConnectorInstallFailureCleansUp(t *testing.T) {
	svc, m := newService(t)
	zipBytes := buildZip(t, map[string]string{
		"info.json":    `{"id":"my_conn","name":"My","runtime":"python"}`,
		"connector.py": "x",
	})

	m.writer.EXPECT().Extract(gomock.Any(), "my_conn", gomock.Any(), "").Return(nil)
	m.installer.EXPECT().Install(gomock.Any(), "my_conn").Return(errors.New("pip exploded"))
	m.writer.EXPECT().Remove(gomock.Any(), "my_conn").Return(nil)

	_, err := svc.UploadConnector(context.Background(), zipBytes, "")

	assert.ErrorContains(t, err, "pip exploded")
}

func TestDeleteConnector(t *testing.T) {
	svc, m := newService(t)
	m.store.EXPECT().Get(gomock.Any(), "my_conn").
		Return(domain.ConnectorInfo{"id": "my_conn"}, nil)
	m.writer.EXPECT().Remove(gomock.Any(), "my_conn").Return(nil)
	// built-ins have no audit row: not-found from the repo is fine
	m.repo.EXPECT().Delete(gomock.Any(), "my_conn").Return(connectors.ErrConnectorNotFound)

	assert.NoError(t, svc.DeleteConnector(context.Background(), "my_conn"))
}

// Reserved ids (builtins, code snippets, core) guard deletes as well as
// uploads: removing their tree dir would strip them from the editor.
func TestDeleteConnectorRejectsReservedIDs(t *testing.T) {
	svc, _ := newService(t) // no store/writer/repo expectations: nothing may be touched

	for _, id := range []string{"core", "code_snippet_py", "code_snippet_js", "condition", "http_request"} {
		err := svc.DeleteConnector(context.Background(), id)
		assert.ErrorIs(t, err, connectors.ErrInvalidConnector, id)
	}
}

func TestDeleteConnectorNotFound(t *testing.T) {
	svc, m := newService(t)
	m.store.EXPECT().Get(gomock.Any(), "ghost").
		Return(nil, connectors.ErrConnectorNotFound)

	err := svc.DeleteConnector(context.Background(), "ghost")

	assert.ErrorIs(t, err, connectors.ErrConnectorNotFound)
}
