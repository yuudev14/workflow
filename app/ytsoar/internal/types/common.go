package types

type Entries[T any] struct {
	Entries []T `json:"entries"`
	Total   int `json:"total"`
}
