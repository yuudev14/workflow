package contracts_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/application/contracts"
)

type prefixBox struct{ failOpen bool }

func (prefixBox) Seal(plain string) (string, error) { return "sealed:" + plain, nil }
func (b prefixBox) Open(sealed string) (string, error) {
	if b.failOpen {
		return "", errors.New("bad key")
	}
	return strings.TrimPrefix(sealed, "sealed:"), nil
}

func TestSecretValues(t *testing.T) {
	box := prefixBox{}

	t.Run("SealValues seals strings and skips the mask and non-strings", func(t *testing.T) {
		out, err := contracts.SealValues(box, json.RawMessage(`{"a":"x","pw":"********","n":3,"e":""}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":"sealed:x","pw":"********","n":3,"e":""}`, string(out))
	})

	t.Run("SealValues rejects a config that is not an object", func(t *testing.T) {
		_, err := contracts.SealValues(box, json.RawMessage(`[1]`))
		assert.Error(t, err)
	})

	t.Run("OpenValues returns the plain values", func(t *testing.T) {
		out, err := contracts.OpenValues(box, json.RawMessage(`{"a":"sealed:x"}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":"x"}`, string(out))
	})

	t.Run("MaskedView masks only the secret keys", func(t *testing.T) {
		out := contracts.MaskedView(box, json.RawMessage(`{"url":"sealed:u","pw":"sealed:p"}`), []string{"pw"})
		assert.JSONEq(t, `{"url":"u","pw":"********"}`, string(out))
	})

	t.Run("MaskedView still masks a value that does not open", func(t *testing.T) {
		out := contracts.MaskedView(prefixBox{failOpen: true}, json.RawMessage(`{"pw":"raw"}`), []string{"pw"})
		assert.JSONEq(t, `{"pw":"********"}`, string(out))
	})

	t.Run("KeepMasked puts the stored value back for a mask", func(t *testing.T) {
		out := contracts.KeepMasked(json.RawMessage(`{"pw":"sealed:old","a":"sealed:1"}`), json.RawMessage(`{"pw":"********","a":"sealed:2"}`))
		assert.JSONEq(t, `{"pw":"sealed:old","a":"sealed:2"}`, string(out))
	})
}
