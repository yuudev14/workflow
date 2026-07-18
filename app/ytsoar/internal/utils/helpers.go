package utils

import (
	"database/sql"
	"encoding/json"

	"github.com/yuudev14/ytsoar/internal/logger"
)

func NullStringToInterface(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Helper
func StrPtr(s string) *string { return &s }

// DebugJSON marshals v to indented JSON and logs it at debug level under
// label, so it shows up on the dev logger's colored DEBUG line.
func DebugJSON(log logger.Logger, label string, v any, indent string) {
	b, err := json.MarshalIndent(v, "", indent)
	if err != nil {
		log.Debugf("%s: failed to marshal to json: %v", label, err)
		return
	}
	log.Debugf("%s:\n%s", label, string(b))
}
