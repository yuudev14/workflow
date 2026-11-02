package domain

import (
	"context"
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

type actorLabelKey struct{}

// WithActorLabel names an actor that is not a user, for example a playbook run. The
// timeline and the notes show it when the actor id is nil.
func WithActorLabel(ctx context.Context, label string) context.Context {
	return context.WithValue(ctx, actorLabelKey{}, label)
}

func ActorLabel(ctx context.Context) *string {
	if label, ok := ctx.Value(actorLabelKey{}).(string); ok {
		return &label
	}
	return nil
}

// FieldChange is one entry in an `updated` timeline event. Usernames are filled in for
// id fields, so the timeline reads as a sentence and not a pair of uuids.
type FieldChange struct {
	Field        string  `json:"field"`
	From         any     `json:"from"`
	To           any     `json:"to"`
	FromUsername *string `json:"from_username,omitempty"`
	ToUsername   *string `json:"to_username,omitempty"`
}

// DiffAssignable builds the timeline diff for the fields that alerts and incidents both
// expose through PATCH. It compares the saved before and after rows, not the request, so a
// PATCH that sets the same value gives no change and does not pollute the audit log.
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

// tag order does not matter, and a reorder is not an edit worth logging, so compare the
// sorted copies
func sameTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}
