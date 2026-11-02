package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upload sends a file in the multipart field "file".
func (c *client) upload(t *testing.T, path string, file []byte) (int, any) {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "connector.zip")
	require.NoError(t, err)
	_, err = part.Write(file)
	require.NoError(t, err)
	require.NoError(t, form.Close())
	res := c.send(t, http.MethodPost, path, &body, form.FormDataContentType())
	defer res.Body.Close()
	var out any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// zipDir zips the files of a connector folder.
func zipDir(t *testing.T, dir string) []byte {
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	require.NoError(t, filepath.WalkDir(dir, func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name, _ := filepath.Rel(dir, file)
		w, err := archive.Create(filepath.ToSlash(name))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}))
	require.NoError(t, archive.Close())
	return buf.Bytes()
}

func TestConnectors(t *testing.T) {
	const base = "/api/v1/connectors"
	c := admin(t)

	t.Run("POST /connectors", func(t *testing.T) {
		t.Run("returns 400 when the file is missing", func(t *testing.T) {
			c.expect(t, http.StatusBadRequest, http.MethodPost, base, nil)
		})
		t.Run("returns 400 when the file is not a zip", func(t *testing.T) {
			code, _ := c.upload(t, base, []byte("not a zip"))
			assert.Equal(t, http.StatusBadRequest, code)
		})
		t.Run("installs a connector from a zip", func(t *testing.T) {
			code, out := c.upload(t, base, zipDir(t, "../../../connectors/sample_js"))
			require.Equal(t, http.StatusCreated, code, out)
		})
		t.Run("returns 409 when the connector exists", func(t *testing.T) {
			code, out := c.upload(t, base, zipDir(t, "../../../connectors/sample_js"))
			assert.Equal(t, http.StatusConflict, code)
			assert.Contains(t, toJSON(out), "sample_js")
		})
		t.Run("replaces the connector with replace=true", func(t *testing.T) {
			code, out := c.upload(t, base+"?replace=true", zipDir(t, "../../../connectors/sample_js"))
			assert.Equal(t, http.StatusCreated, code, out)
		})
	})

	t.Run("GET /connectors", func(t *testing.T) {
		t.Run("lists the connector", func(t *testing.T) {
			assert.Contains(t, ids(c.get(t, base)), "sample_js")
		})
		t.Run("returns the connector", func(t *testing.T) {
			c.get(t, base+"/sample_js")
		})
		t.Run("returns 404 when the connector does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/unknown", nil)
		})
	})

	t.Run("/connectors/:id/configs", func(t *testing.T) {
		code, out := c.upload(t, base+"?replace=true", zipDir(t, "../../../connectors/sample"))
		require.Equal(t, http.StatusCreated, code, out)
		configs := base + "/sample/configs"
		prod := c.create(t, configs, J{"name": unique("prod"), "config": J{"server_url": "https://prod", "api_key": "secret"}})

		t.Run("masks the password field and makes the first config the default", func(t *testing.T) {
			assert.Equal(t, true, prod["is_default"])
			assert.Equal(t, J{"server_url": "https://prod", "api_key": "********"}, prod["config"])
		})
		t.Run("stores every value encrypted", func(t *testing.T) {
			var stored string
			require.NoError(t, testPool.QueryRow(context.Background(),
				"SELECT config::text FROM connector_configs WHERE id = $1", prod["id"]).Scan(&stored))
			assert.NotContains(t, stored, "secret")
			assert.NotContains(t, stored, "https://prod")
		})
		t.Run("moves the default to a new default config", func(t *testing.T) {
			lab := c.create(t, configs, J{"name": unique("lab"), "is_default": true})
			defaults := 0
			for _, item := range items(c.get(t, configs)) {
				if item.(J)["is_default"] == true {
					defaults++
					assert.Equal(t, lab["id"], item.(J)["id"])
				}
			}
			assert.Equal(t, 1, defaults)
		})
		c.expectAll(t, []request{
			{"returns 409 when the name exists", http.MethodPost, configs, J{"name": prod["name"]}, http.StatusConflict},
			{"returns 400 when the config is not an object", http.MethodPost, configs, J{"name": unique("x"), "config": "text"}, http.StatusBadRequest},
			{"returns 404 when the connector does not exist", http.MethodGet, base + "/unknown/configs", nil, http.StatusNotFound},
			{"updates a config", http.MethodPatch, configs + "/" + prod["id"].(string), J{"config": J{"server_url": "https://new", "api_key": "********"}}, http.StatusOK},
			{"deletes a config", http.MethodDelete, configs + "/" + prod["id"].(string), nil, http.StatusNoContent},
			{"returns 404 when the config does not exist", http.MethodDelete, configs + "/" + prod["id"].(string), nil, http.StatusNotFound},
		})
	})

	t.Run("DELETE /connectors/:id", func(t *testing.T) {
		t.Run("deletes the connector", func(t *testing.T) {
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/sample_js", nil)
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/sample_js", nil)
		})
		t.Run("returns 404 when the connector does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodDelete, base+"/sample_js", nil)
		})
	})
}
