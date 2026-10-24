package integration

import (
	"archive/zip"
	"bytes"
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
			code, _ := c.upload(t, base, zipDir(t, "../../../connectors/sample_js"))
			assert.Equal(t, http.StatusConflict, code)
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
