package domain

type EdgeHandle struct {
	SourceHandle      *string `json:"source_handle"`
	DestinationHandle *string `json:"destination_handle,omitempty"`
}
