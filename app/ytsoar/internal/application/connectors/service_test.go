package connectors_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/application/connectors/mocks"
	contractmocks "github.com/yuudev14/ytsoar/internal/application/contracts/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
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

	info, err := svc.UploadConnector(context.Background(), zipBytes, "alice", true)

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

	_, err := svc.UploadConnector(context.Background(), zipBytes, "", true)

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
			_, err := svc.UploadConnector(context.Background(), buildZip(t, tc.files), "", true)
			assert.ErrorIs(t, err, connectors.ErrInvalidConnector)
		})
	}
}

func TestUploadConnectorNotAZip(t *testing.T) {
	svc, _ := newService(t)

	_, err := svc.UploadConnector(context.Background(), []byte("plain text"), "", true)

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

	_, err := svc.UploadConnector(context.Background(), zipBytes, "", true)

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

// Reserved ids (builtins, code snippets, core) guard deletes as well as uploads: removing
// their tree dir would remove them from the editor.
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

// prefixBox is a readable SecretBox for tests.
type prefixBox struct{}

func (prefixBox) Seal(plain string) (string, error) { return "sealed:" + plain, nil }
func (prefixBox) Open(sealed string) (string, error) {
	if plain, ok := strings.CutPrefix(sealed, "sealed:"); ok {
		return plain, nil
	}
	return "", errors.New("not sealed")
}

var configInfo = domain.ConnectorInfo{"id": "abuseipdb", "configuration": map[string]any{"fields": []any{
	map[string]any{"name": "base_url", "type": "text"},
	map[string]any{"name": "api_key", "type": "password"},
}}}

type configMocks struct {
	store *mocks.MockConnectorStore
	repo  *mocks.MockConfigRepository
}

func newConfigService(t *testing.T) (*connectors.ConfigServiceImpl, configMocks) {
	ctrl := gomock.NewController(t)
	m := configMocks{store: mocks.NewMockConnectorStore(ctrl), repo: mocks.NewMockConfigRepository(ctrl)}
	tx := contractmocks.NewMockTxManager(ctrl)
	tx.EXPECT().WithinTransaction(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	audit := contractmocks.NewMockAuditWriter(ctrl)
	audit.EXPECT().Insert(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	m.store.EXPECT().Get(gomock.Any(), "abuseipdb").AnyTimes().Return(configInfo, nil)
	return connectors.NewConfigService(logger.NewNop(), m.store, m.repo, prefixBox{}, tx, audit), m
}

func TestConfigService(t *testing.T) {
	t.Run("CreateConfig", func(t *testing.T) {
		t.Run("seals each string value, masks the password and makes the first config the default", func(t *testing.T) {
			svc, m := newConfigService(t)
			m.repo.EXPECT().ListConfigs(gomock.Any(), "abuseipdb").Return(nil, nil)
			m.repo.EXPECT().ClearDefault(gomock.Any(), "abuseipdb").Return(nil)
			m.repo.EXPECT().CreateConfig(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, c domain.ConnectorConfig) (domain.ConnectorConfig, error) {
					assert.JSONEq(t, `{"base_url":"sealed:https://x","api_key":"sealed:k","verify":true}`, string(c.Config))
					return c, nil
				})

			got, err := svc.CreateConfig(context.Background(), nil, "abuseipdb", connectors.ConfigInput{
				Name: "prod", Config: json.RawMessage(`{"base_url":"https://x","api_key":"k","verify":true}`),
			})
			require.NoError(t, err)
			assert.True(t, got.IsDefault)
			assert.JSONEq(t, `{"base_url":"https://x","api_key":"********","verify":true}`, string(got.Config))
		})

		t.Run("rejects a bad name", func(t *testing.T) {
			svc, _ := newConfigService(t)
			_, err := svc.CreateConfig(context.Background(), nil, "abuseipdb", connectors.ConfigInput{Name: "../x"})
			kind, _ := apperr.KindOf(err)
			assert.Equal(t, apperr.Invalid, kind)
		})
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		t.Run("keeps the stored secret when the value is the mask", func(t *testing.T) {
			svc, m := newConfigService(t)
			id := uuid.New()
			m.repo.EXPECT().GetConfig(gomock.Any(), "abuseipdb", id).Return(domain.ConnectorConfig{
				ID: id, Config: json.RawMessage(`{"base_url":"sealed:old","api_key":"sealed:secret"}`),
			}, nil)
			m.repo.EXPECT().UpdateConfig(gomock.Any(), "abuseipdb", id, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, _ uuid.UUID, u connectors.ConfigUpdate) (domain.ConnectorConfig, error) {
					assert.JSONEq(t, `{"base_url":"sealed:new","api_key":"sealed:secret"}`, string(u.Config))
					return domain.ConnectorConfig{ID: id, Config: u.Config}, nil
				})

			_, err := svc.UpdateConfig(context.Background(), nil, "abuseipdb", id, connectors.UpdateConfigInput{
				Config: json.RawMessage(`{"base_url":"new","api_key":"********"}`),
			})
			require.NoError(t, err)
		})
	})

	t.Run("RunConfig", func(t *testing.T) {
		t.Run("opens the values of the default config", func(t *testing.T) {
			svc, m := newConfigService(t)
			m.repo.EXPECT().GetConfigForRun(gomock.Any(), "abuseipdb", (*string)(nil)).Return(domain.ConnectorConfig{
				Config: json.RawMessage(`{"api_key":"sealed:secret","verify":false}`),
			}, nil)

			values, err := svc.RunConfig(context.Background(), "abuseipdb", new(""))
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"api_key": "secret", "verify": false}, values)
		})

		t.Run("gives an empty config when the connector has no default", func(t *testing.T) {
			svc, m := newConfigService(t)
			m.repo.EXPECT().GetConfigForRun(gomock.Any(), "abuseipdb", (*string)(nil)).Return(domain.ConnectorConfig{}, connectors.ErrConfigNotFound)

			values, err := svc.RunConfig(context.Background(), "abuseipdb", nil)
			require.NoError(t, err)
			assert.Empty(t, values)
		})

		t.Run("fails when the named config does not exist", func(t *testing.T) {
			svc, m := newConfigService(t)
			m.repo.EXPECT().GetConfigForRun(gomock.Any(), "abuseipdb", gomock.Any()).Return(domain.ConnectorConfig{}, connectors.ErrConfigNotFound)

			_, err := svc.RunConfig(context.Background(), "abuseipdb", new("lab"))
			assert.ErrorIs(t, err, connectors.ErrConfigNotFound)
		})
	})
}
