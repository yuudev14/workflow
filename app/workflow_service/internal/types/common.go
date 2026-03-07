package types

type Entries[T interface{}] struct {
	Entries []T `json:"entries"`
	Total   int `json:"total"`
}
