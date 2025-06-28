package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/pkg/utils"
)

func TestSet(t *testing.T) {
	set := make(utils.Set[int])

	tests := []struct {
		method   func(item int)
		param    int
		expected []int
	}{
		{
			method:   set.Add,
			param:    1,
			expected: []int{1},
		},
		{
			method:   set.Add,
			param:    2,
			expected: []int{1, 2},
		},
		{
			method:   set.Remove,
			param:    1,
			expected: []int{2},
		},
	}

	for _, test := range tests {
		test.method(test.param)
		assert.Equal(t, set.ToList(), test.expected)
	}
}

func TestSetIn(t *testing.T) {
	set := make(utils.Set[int])
	set.Add(1)
	set.Add(2)

	tests := []struct {
		param    int
		expected bool
	}{
		{
			param:    1,
			expected: true,
		},
		{
			param:    2,
			expected: true,
		},
		{
			param:    3,
			expected: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, set.Has(test.param), test.expected)
	}
}
