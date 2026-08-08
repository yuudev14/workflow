package domain

import (
	"slices"

	"github.com/google/uuid"
)

const (
	ModuleEventAlert    = "alert"
	ModuleEventIncident = "incident"
)

const (
	ModuleEventCreated = "created"
	ModuleEventUpdated = "updated"
)

// FieldChange is one entry in an `updated` timeline event. Usernames are filled
// in for id-valued fields so the timeline reads as a sentence rather than a pair
// of uuids.
type FieldChange struct {
	Field        string  `json:"field"`
	From         any     `json:"from"`
	To           any     `json:"to"`
	FromUsername *string `json:"from_username,omitempty"`
	ToUsername   *string `json:"to_username,omitempty"`
}

// DiffAssignable computes the timeline diff for the fields both alerts and
// incidents expose through PATCH. Comparing the persisted before/after rows
// rather than the request payload means a PATCH that sets a field to the value
// it already held produces no change - a no-op edit must not pollute the audit
// log.
func DiffAssignable(
	beforeSeverity, afterSeverity AlertSeverity,
	beforeAssignee, afterAssignee *uuid.UUID,
	beforeTeam, afterTeam *uuid.UUID,
	beforeTags, afterTags []string,
) []FieldChange {
	changes := make([]FieldChange, 0, 4)

	if beforeSeverity != afterSeverity {
		changes = append(changes, FieldChange{Field: "severity", From: beforeSeverity, To: afterSeverity})
	}
	if !sameUUID(beforeAssignee, afterAssignee) {
		changes = append(changes, FieldChange{
			Field: "assignee_id", From: uuidOrNil(beforeAssignee), To: uuidOrNil(afterAssignee),
		})
	}
	if !sameUUID(beforeTeam, afterTeam) {
		changes = append(changes, FieldChange{
			Field: "team_id", From: uuidOrNil(beforeTeam), To: uuidOrNil(afterTeam),
		})
	}
	if !sameTags(beforeTags, afterTags) {
		changes = append(changes, FieldChange{Field: "tags", From: beforeTags, To: afterTags})
	}

	return changes
}

func sameUUID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func uuidOrNil(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}

// Tag order is not meaningful, but reordering is also not an edit worth logging,
// so the comparison is on the sorted copies.
func sameTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}
