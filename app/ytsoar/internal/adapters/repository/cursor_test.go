// Internal test package: the cursor helpers are unexported by design - nothing
// outside the repository layer should be able to mint a token.
package repository

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

func TestCursorRoundTrip(t *testing.T) {
	id := uuid.New()
	at := time.Date(2026, 7, 25, 10, 12, 3, 441000000, time.UTC)

	token, err := encodeCursor("created_at", at, id)
	require.NoError(t, err)

	got, err := decodeCursor(token)
	require.NoError(t, err)

	assert.Equal(t, "created_at", got.Sort)
	assert.Equal(t, "desc", got.Dir)
	assert.Equal(t, id, got.ID)
	assert.True(t, got.Value.Equal(at), "want %s, got %s", at, got.Value)
}

// A caller in another timezone must not shift the keyset predicate.
func TestCursorNormalizesToUTC(t *testing.T) {
	zone := time.FixedZone("UTC+9", 9*60*60)
	at := time.Date(2026, 7, 25, 19, 12, 3, 0, zone)

	token, err := encodeCursor("created_at", at, uuid.New())
	require.NoError(t, err)

	got, err := decodeCursor(token)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Value.Location())
	assert.True(t, got.Value.Equal(at))
}

func TestDecodeCursorRejectsBadTokens(t *testing.T) {
	valid, err := encodeCursor("created_at", time.Now(), uuid.New())
	require.NoError(t, err)

	tamper := func(mutate func(*pageCursor)) string {
		raw, decodeErr := base64.RawURLEncoding.DecodeString(valid)
		require.NoError(t, decodeErr)
		var c pageCursor
		require.NoError(t, json.Unmarshal(raw, &c))
		mutate(&c)
		out, marshalErr := json.Marshal(c)
		require.NoError(t, marshalErr)
		return base64.RawURLEncoding.EncodeToString(out)
	}

	cases := []struct {
		name  string
		token string
	}{
		{"garbage", "not-a-cursor!!"},
		{"truncated", valid[:len(valid)/2]},
		{"empty", ""},
		{"valid base64, not json", base64.RawURLEncoding.EncodeToString([]byte("hello"))},
		// The whole point of the whitelist: a sort key becomes raw SQL in
		// OrderBy, so an unknown column must never reach the query builder.
		{"non-whitelisted sort", tamper(func(c *pageCursor) { c.Sort = "reporter" })},
		{"sql injection in sort", tamper(func(c *pageCursor) { c.Sort = "created_at; DROP TABLE alerts" })},
		{"unsupported direction", tamper(func(c *pageCursor) { c.Dir = "asc" })},
		{"nil id", tamper(func(c *pageCursor) { c.ID = uuid.Nil })},
		{"zero value", tamper(func(c *pageCursor) { c.Value = time.Time{} })},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeCursor(tc.token)
			require.Error(t, err)
			kind, _ := apperr.KindOf(err)
			assert.Equal(t, apperr.Invalid, kind,
				"a bad cursor is a client error, never a 500")
		})
	}
}

func TestEncodeCursorRejectsNonWhitelistedSort(t *testing.T) {
	_, err := encodeCursor("reporter", time.Now(), uuid.New())
	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
}

// The row-value comparison is the whole reason the cursor carries an id: NOW()
// is fixed at transaction start, so a batch insert gives every row the same
// created_at and a timestamp-only predicate cannot page past it.
func TestApplyKeysetBuildsRowValuePredicate(t *testing.T) {
	id := uuid.New()
	at := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	token, err := encodeCursor("created_at", at, id)
	require.NoError(t, err)

	stmt, err := applyKeyset(
		sq.Select("a.id").From("alerts a").PlaceholderFormat(sq.Dollar), "a", &token)
	require.NoError(t, err)

	sqlStr, args, err := orderKeyset(stmt, "a").ToSql()
	require.NoError(t, err)

	assert.Contains(t, sqlStr, "(a.created_at, a.id) < ($1, $2)")
	assert.Contains(t, sqlStr, "ORDER BY a.created_at DESC, a.id DESC")
	require.Len(t, args, 2)
	assert.Equal(t, id, args[1])
}

func TestApplyKeysetWithoutCursorIsUnfiltered(t *testing.T) {
	base := sq.Select("a.id").From("alerts a").PlaceholderFormat(sq.Dollar)
	empty := ""

	for _, token := range []*string{nil, &empty} {
		stmt, err := applyKeyset(base, "a", token)
		require.NoError(t, err)

		sqlStr, args, err := stmt.ToSql()
		require.NoError(t, err)
		assert.NotContains(t, sqlStr, "WHERE")
		assert.Empty(t, args)
	}
}

func TestApplyKeysetPropagatesDecodeFailure(t *testing.T) {
	bad := "not-a-cursor!!"
	_, err := applyKeyset(sq.Select("a.id").From("alerts a"), "a", &bad)
	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
}
