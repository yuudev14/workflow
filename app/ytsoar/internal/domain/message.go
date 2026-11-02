package domain

import "github.com/google/uuid"

// TaskMessage is the wire format published to the playbook queue when a playbook is
// triggered. The graph is an adjacency list keyed by task name. Edges is extra metadata
// for conditional branching: absent or empty means every edge is followed (old messages
// keep working).
type TaskMessage struct {
	Graph             map[string][]string `json:"graph"`
	Tasks             map[string]Tasks    `json:"tasks"`
	PlaybookHistoryId uuid.UUID           `json:"playbook_history_id"`
	PlaybookName      string              `json:"playbook_name,omitempty"`
	Edges             []EdgeRef           `json:"edges,omitempty"`
	// Input is the trigger data, available in templates as `var.input`. It is optional and
	// additive, the only change this wire format allows. An old producer that omits it still
	// runs.
	Input *RunInput `json:"input,omitempty"`
}

// EdgeRef is one playbook edge on the wire. A SourceHandle of "true" or "false" is a
// conditional branch (followed only when the output {"result": bool} of the source node
// matches). Any other value always follows.
type EdgeRef struct {
	Source       string  `json:"source"`
	Destination  string  `json:"destination"`
	SourceHandle *string `json:"source_handle,omitempty"`
}
