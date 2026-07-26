package types

type CursorPage[T any] struct {
	Entries    []T     `json:"entries"`
	Total      int     `json:"total"`
	NextCursor *string `json:"next_cursor,omitempty"`
}
