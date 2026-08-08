package repository

import (
	"encoding/base64"
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

var errInvalidCursor = apperr.New(apperr.Invalid, "invalid cursor")

// sortableColumns gates which column a cursor may name. A sort key reaching
// squirrel's OrderBy is raw SQL, so anything not in this map is an injection
// vector, not a 404.
var sortableColumns = map[string]bool{
	"created_at": true,
}

// pageCursor is deliberately opaque on the wire: callers hand back the token
// verbatim, which lets a new sort ship as one map entry plus one index instead
// of an API change.
type pageCursor struct {
	Sort  string    `json:"s"`
	Dir   string    `json:"d"`
	Value time.Time `json:"v"`
	ID    uuid.UUID `json:"i"`
}

func encodeCursor(sort string, value time.Time, id uuid.UUID) (string, error) {
	if !sortableColumns[sort] {
		return "", errInvalidCursor
	}

	raw, err := json.Marshal(pageCursor{Sort: sort, Dir: "desc", Value: value.UTC(), ID: id})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeCursor(token string) (pageCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return pageCursor{}, errInvalidCursor
	}

	var c pageCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return pageCursor{}, errInvalidCursor
	}
	if !sortableColumns[c.Sort] || c.Dir != "desc" || c.ID == uuid.Nil || c.Value.IsZero() {
		return pageCursor{}, errInvalidCursor
	}
	return c, nil
}

// applyKeyset compares (sort, id) as a row value rather than the timestamp
// alone: NOW() is fixed at transaction start, so a batch insert gives every row
// an identical created_at and a timestamp-only predicate cannot page past it.
func applyKeyset(stmt sq.SelectBuilder, alias string, token *string) (sq.SelectBuilder, error) {
	if token == nil || *token == "" {
		return stmt, nil
	}

	c, err := decodeCursor(*token)
	if err != nil {
		return stmt, err
	}

	col := alias + "." + c.Sort
	return stmt.Where(sq.Expr("("+col+", "+alias+".id) < (?, ?)", c.Value, c.ID)), nil
}

func orderKeyset(stmt sq.SelectBuilder, alias string) sq.SelectBuilder {
	return stmt.OrderBy(alias+".created_at DESC", alias+".id DESC")
}
