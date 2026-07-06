package goconnectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ConditionConnector powers conditional branching: it compares two (already
// templated) operands and returns {"result": bool}. The executor follows the
// outgoing edge whose source_handle ("true"/"false") matches that result.
type ConditionConnector struct{}

func NewConditionConnector() *ConditionConnector {
	return &ConditionConnector{}
}

func (c *ConditionConnector) Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error) {
	if operation != "evaluate" {
		return nil, fmt.Errorf("operation (%s) does not exist in ConditionConnector", operation)
	}
	operator, _ := params["operator"].(string)
	result, err := compare(params["left"], operator, params["right"])
	if err != nil {
		return nil, err
	}
	return map[string]any{"result": result}, nil
}

// compare works on templated values, which arrive as strings: ordering
// operators need both sides numeric; equality falls back to numeric
// comparison when both sides parse, so "5" == "5.0" holds.
func compare(left any, operator string, right any) (bool, error) {
	l, r := asString(left), asString(right)
	lNum, lOk := asNumber(left)
	rNum, rOk := asNumber(right)
	numeric := lOk && rOk

	switch operator {
	case "==":
		if numeric {
			return lNum == rNum, nil
		}
		return l == r, nil
	case "!=":
		if numeric {
			return lNum != rNum, nil
		}
		return l != r, nil
	case ">", "<", ">=", "<=":
		if !numeric {
			return false, fmt.Errorf("operator %q needs numeric operands, got %q and %q", operator, l, r)
		}
		switch operator {
		case ">":
			return lNum > rNum, nil
		case "<":
			return lNum < rNum, nil
		case ">=":
			return lNum >= rNum, nil
		default:
			return lNum <= rNum, nil
		}
	case "contains":
		return strings.Contains(l, r), nil
	case "not_contains":
		return !strings.Contains(l, r), nil
	default:
		return false, fmt.Errorf("unknown operator %q", operator)
	}
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, err == nil
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}
