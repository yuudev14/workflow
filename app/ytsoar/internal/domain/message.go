package domain

import "github.com/google/uuid"

// TaskMessage is the wire format published to the playbook queue when a
// playbook is triggered. The graph is an adjacency list keyed by task name.
// Edges is additive metadata for conditional branching: absent or empty means
// every edge is followed unconditionally (older messages keep working).
type TaskMessage struct {
	Graph             map[string][]string `json:"graph"`
	Tasks             map[string]Tasks    `json:"tasks"`
	PlaybookHistoryId uuid.UUID           `json:"playbook_history_id"`
	Edges             []EdgeRef           `json:"edges,omitempty"`
}

// EdgeRef mirrors one playbook edge on the wire. A SourceHandle of "true" or
// "false" marks a conditional branch (followed only when the source node's
// output {"result": bool} matches); any other value follows unconditionally.
type EdgeRef struct {
	Source       string  `json:"source"`
	Destination  string  `json:"destination"`
	SourceHandle *string `json:"source_handle,omitempty"`
}
