package domain

const (
	// ConditionConnectorID is the switch connector for conditional branching.
	ConditionConnectorID = "condition"

	// ConditionOutputHandle is the single output handle of a condition node in the editor
	// (frontend CONDITION_OUTPUT_HANDLE). Edges that still have it were never routed to a
	// branch, so the executor does not follow them.
	ConditionOutputHandle = "output"
)

type EdgeHandle struct {
	SourceHandle      *string `json:"source_handle"`
	DestinationHandle *string `json:"destination_handle,omitempty"`
}
