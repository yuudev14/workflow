package goconnectors

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ConditionConnector powers conditional branching. Both operations are a switch:
// an ordered list of cases, each with a stable "id" naming its branch. The first
// case that matches wins -> {"result": "<id>"}; none match -> {"result": "else"}.
// The editor routes each id to an outgoing edge, which the executor then follows.
// They differ only in how a case is written:
//   - "switch": simple left/operator/right compare (==, !=, contains, ...) — no
//     templating to learn.
//   - "switch_expression": the advanced mode — a full template expression per case,
//     already rendered by the registry (gonja/jinja2), so {{ ...score > 80 }}
//     arrives as "True" and is read for truthiness.
type ConditionConnector struct{}

func NewConditionConnector() *ConditionConnector {
	return &ConditionConnector{}
}

func (c *ConditionConnector) Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error) {
	switch operation {
	case "switch":
		result, err := evaluateSwitchSimple(params["cases"])
		if err != nil {
			return nil, err
		}
		return map[string]any{"result": result}, nil
	case "switch_expression":
		result, err := evaluateSwitch(params["cases"])
		if err != nil {
			return nil, err
		}
		return map[string]any{"result": result}, nil
	default:
		return nil, fmt.Errorf("operation (%s) does not exist in ConditionConnector", operation)
	}
}

// evaluateSwitchSimple walks the ordered cases (each {id, left, operator, right})
// and returns the id of the first whose compare holds, or "else". A compare error
// (e.g. ">" on non-numeric) counts as no match, so one bad case never fails the run.
func evaluateSwitchSimple(rawCases any) (string, error) {
	cases, ok := rawCases.([]any)
	if !ok {
		return "else", nil
	}
	if err := validateCaseIDs(cases); err != nil {
		return "", err
	}
	for _, raw := range cases {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		operator, _ := c["operator"].(string)
		if match, err := compare(c["left"], operator, c["right"]); err == nil && match {
			id, _ := c["id"].(string)
			return id, nil
		}
	}
	return "else", nil
}

// evaluateSwitch walks the ordered cases (each {id, expression}, expression
// already rendered by the registry) and returns the id of the first truthy one,
// or "else".
func evaluateSwitch(rawCases any) (string, error) {
	cases, ok := rawCases.([]any)
	if !ok {
		return "else", nil
	}
	if err := validateCaseIDs(cases); err != nil {
		return "", err
	}
	for _, raw := range cases {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if truthy(c["expression"]) {
			id, _ := c["id"].(string)
			return id, nil
		}
	}
	return "else", nil
}

// validateCaseIDs requires every case to carry a stable id. A positional
// fallback would silently misroute edges when cases are reordered or deleted,
// so a missing id fails the node loudly instead.
func validateCaseIDs(cases []any) error {
	for i, raw := range cases {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := c["id"].(string); !ok || id == "" {
			return fmt.Errorf("case %d has no id", i)
		}
	}
	return nil
}

// truthy reads a rendered template value as true or false. Templates come back as
// strings ("True", "", "0", "None", ...), so we treat the usual empty/zero/none
// forms as false and anything else as true.
func truthy(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case nil:
		return false
	case float64:
		return v != 0
	case int:
		return v != 0
	case string:
		s := strings.TrimSpace(strings.ToLower(v))
		switch s {
		case "", "false", "none", "nil", "null", "no", "off", "[]", "{}":
			return false
		}
		// catch every rendered zero form ("0", "0.0", "0.00", "-0") numerically
		if n, err := strconv.ParseFloat(s, 64); err == nil && n == 0 {
			return false
		}
		return true
	default:
		return true
	}
}

// canonicalNumber matches plainly-formatted numbers ("5", "-3.25") — no
// leading zeros, no exponent. Equality coerces to numbers only for these, so
// "5" == "5.0" holds but id-like values ("0123", "1e3") compare as strings
// and never collide numerically.
var canonicalNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// compare works on templated values, which arrive as strings: ordering
// operators need both sides numeric; equality compares numerically only when
// both sides are canonical numbers, otherwise as strings.
func compare(left any, operator string, right any) (bool, error) {
	l, r := asString(left), asString(right)
	lNum, lOk := asNumber(left)
	rNum, rOk := asNumber(right)
	numeric := lOk && rOk
	eqNumeric := numeric &&
		canonicalNumber.MatchString(strings.TrimSpace(l)) &&
		canonicalNumber.MatchString(strings.TrimSpace(r))

	switch operator {
	case "==":
		if eqNumeric {
			return lNum == rNum, nil
		}
		return l == r, nil
	case "!=":
		if eqNumeric {
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
