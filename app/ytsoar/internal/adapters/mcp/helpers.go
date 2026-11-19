package mcp

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

func runSummary(run domain.PlaybookHistoryResponse) J {
	return J{
		"id":           run.ID,
		"playbook_id":  run.PlaybookID,
		"status":       run.Status,
		"error":        run.Error,
		"trigger_type": run.TriggerType,
		"triggered_at": run.TriggeredAt,
		"finished_at":  run.FinishedAt,
	}
}

func runFilter(in listRunsIn) (playbooks.PlaybookHistoryFilter, error) {
	filter := playbooks.PlaybookHistoryFilter{}
	if in.PlaybookID != "" {
		if _, err := parseID("playbook_id", in.PlaybookID); err != nil {
			return filter, err
		}
		filter.PlaybookID = &in.PlaybookID
	}
	if in.RecordID != "" {
		if _, err := parseID("record_id", in.RecordID); err != nil {
			return filter, err
		}
		filter.RecordID = &in.RecordID
	}
	if in.ModuleType != "" {
		if in.ModuleType != "alert" && in.ModuleType != "incident" {
			return filter, apperr.New(apperr.Invalid, "module_type must be alert or incident")
		}
		filter.ModuleType = &in.ModuleType
	}
	return filter, nil
}

func parseID(field, value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperr.New(apperr.Invalid, field+" must be a uuid")
	}
	return id, nil
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return min(limit, maxLimit)
}

func oneOrNone(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func textOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// pick keeps only the given keys of each object in a JSON array.
func pick(raw *json.RawMessage, keys ...string) []J {
	rows := []J{}
	if raw == nil {
		return rows
	}
	var objects []J
	if err := json.Unmarshal(*raw, &objects); err != nil {
		return rows
	}
	for _, object := range objects {
		row := J{}
		for _, key := range keys {
			row[key] = object[key]
		}
		rows = append(rows, row)
	}
	return rows
}

// clip cuts a long result, because one task can return a whole API response.
func clip(raw *json.RawMessage) string {
	if raw == nil {
		return ""
	}
	text := string(*raw)
	if len(text) <= maxResultLen {
		return text
	}
	return text[:maxResultLen] + "... (cut)"
}
