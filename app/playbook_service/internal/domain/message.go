package domain

import "github.com/google/uuid"

// TaskMessage is the wire format published to the playbook queue when a
// playbook is triggered. The graph is an adjacency list keyed by task name.
type TaskMessage struct {
	Graph             map[string][]string `json:"graph"`
	Tasks             map[string]Tasks    `json:"tasks"`
	PlaybookHistoryId uuid.UUID           `json:"playbook_history_id"`
}
