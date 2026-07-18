package types

import (
	"encoding/json"
	"time"
)

type NullableType interface {
	any | bool | string | float64 | int16 | int32 | int64 | time.Time
}
type Nullable[T any] struct {
	Value *T
	Set   bool
}

// If this method was called, the value was set.
func (i *Nullable[T]) UnmarshalJSON(data []byte) error {

	i.Set = true
	var temp *T
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	i.Value = temp

	return nil
}

func (i *Nullable[T]) ToNullableAny() Nullable[any] {
	value := any(i.Value)
	return Nullable[any]{Value: &value, Set: i.Set}
}

type JsonType map[string]interface{}

func (pc *JsonType) Scan(val interface{}) error {
	if val == nil {
		*pc = nil // Explicitly set to nil if value is null
		return nil
	}

	var raw []byte

	switch v := val.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		raw = nil
	}

	return json.Unmarshal(raw, pc)
}
