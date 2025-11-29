package types_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
)

type TestStruct struct {
	Name types.Nullable[sql.NullString] `json:"name"`
}

func TestNullableType(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{
			value:    `{}`,
			expected: false,
		},
		{
			value:    `{"name": null}`,
			expected: true,
		},
	}

	for _, test := range tests {
		var testVal TestStruct
		json.Unmarshal([]byte(test.value), &testVal)
		assert.Equal(t, test.expected, testVal.Name.Set)
	}
}
