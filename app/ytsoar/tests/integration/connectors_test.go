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
	part.Write(file)
	require.NoError(t, form.Close())
	res := c.send(t, http.MethodPost, path, &body, form.FormDataContentType())
	defer res.Body.Close()
	var out any
	json.NewDecoder(res.Body).Decode(&out)
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
	const base = "/api/connectors/v1"
	c := admin(t)

	t.Run("upload", func(t *testing.T) {
		c.expect(t, http.StatusBadRequest, http.MethodPost, base, nil)
		code, _ := c.upload(t, base, []byte("not a zip"))
		assert.Equal(t, http.StatusBadRequest, code, "not a zip")

		code, out := c.upload(t, base, zipDir(t, "../../../connectors/sample_js"))
		require.Equal(t, http.StatusCreated, code, out)
	})

	t.Run("read", func(t *testing.T) {
		assert.Contains(t, ids(c.get(t, base)), "sample_js")
		c.get(t, base+"/sample_js")
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/unknown", nil)
	})

	t.Run("delete", func(t *testing.T) {
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/sample_js", nil)
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/sample_js", nil)
		c.expect(t, http.StatusNotFound, http.MethodDelete, base+"/sample_js", nil)
	})
}
