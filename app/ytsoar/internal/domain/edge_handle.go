package domain

const (
	// ConditionConnectorID is the builtin switch connector powering
	// conditional branching.
	ConditionConnectorID = "condition"

	// ConditionOutputHandle is the editor's single output handle on a
	// condition node (frontend CONDITION_OUTPUT_HANDLE). Edges still carrying
	// it were never routed to a branch, so the executor does not follow them.
	ConditionOutputHandle = "output"
)

type EdgeHandle struct {
	SourceHandle      *string `json:"source_handle"`
	DestinationHandle *string `json:"destination_handle,omitempty"`
}
