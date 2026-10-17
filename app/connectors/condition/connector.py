import re

from connectors.core.connector import Connector

# only plain numbers compare as numbers for ==, so ids like "0123" or "1e3" stay text
CANONICAL_NUMBER = re.compile(r"^-?(0|[1-9][0-9]*)(\.[0-9]+)?$")
FALSY_TEXT = {"", "false", "none", "nil", "null", "no", "off", "[]", "{}"}


class Condition(Connector):
    """A switch: the id of the first matching case wins, else "else".

    "switch" compares left, operator and right. "switch_expression" reads each
    expression, which evaluate_params already rendered with jinja2.
    """

    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        cases = params.get("cases")
        if not isinstance(cases, list):
            cases = []
        for i, c in enumerate(cases):
            # a position fallback would route edges wrong when cases move
            if isinstance(c, dict) and (not isinstance(c.get("id"), str) or not c["id"]):
                raise ValueError(f"case {i} has no id")
        cases = [c for c in cases if isinstance(c, dict)]

        if operation == "switch":
            match = self._first(cases, self._compare_case)
        elif operation == "switch_expression":
            match = self._first(cases, lambda c: truthy(c.get("expression")))
        else:
            raise ValueError(f"operation ({operation}) does not exist in Condition")
        return {"result": match}

    def health_check(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return operation

    @staticmethod
    def _first(cases, matches) -> str:
        for c in cases:
            if matches(c):
                return c["id"]
        return "else"

    @staticmethod
    def _compare_case(c: dict) -> bool:
        # one bad case is no match, so it does not fail the run
        try:
            return compare(c.get("left"), c.get("operator"), c.get("right"))
        except ValueError:
            return False


def truthy(value) -> bool:
    if isinstance(value, str):
        s = value.strip().lower()
        if s in FALSY_TEXT:
            return False
        number = as_number(s)
        return number is None or number != 0
    return bool(value) if isinstance(value, (bool, int, float)) or value is None else True


def as_string(value) -> str:
    if value is None:
        return ""
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, float) and value.is_integer():
        return str(int(value))
    return str(value)


def as_number(value):
    if isinstance(value, bool):
        return 1.0 if value else 0.0
    if isinstance(value, (int, float)):
        return float(value)
    if isinstance(value, str):
        try:
            return float(value.strip())
        except ValueError:
            return None
    return None


def compare(left, operator, right) -> bool:
    l, r = as_string(left), as_string(right)
    l_num, r_num = as_number(left), as_number(right)
    numeric = l_num is not None and r_num is not None
    eq_numeric = (
        numeric
        and CANONICAL_NUMBER.match(l.strip()) is not None
        and CANONICAL_NUMBER.match(r.strip()) is not None
    )

    if operator == "==":
        return l_num == r_num if eq_numeric else l == r
    if operator == "!=":
        return l_num != r_num if eq_numeric else l != r
    if operator in (">", "<", ">=", "<="):
        if not numeric:
            raise ValueError(f"operator {operator!r} needs numeric operands, got {l!r} and {r!r}")
        return {
            ">": l_num > r_num,
            "<": l_num < r_num,
            ">=": l_num >= r_num,
            "<=": l_num <= r_num,
        }[operator]
    if operator == "contains":
        return r in l
    if operator == "not_contains":
        return r not in l
    raise ValueError(f"unknown operator {operator!r}")
