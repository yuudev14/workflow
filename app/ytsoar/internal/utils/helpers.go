package utils

import (
	"database/sql"
)

func NullStringToInterface(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Helper
func StrPtr(s string) *string { return &s }
