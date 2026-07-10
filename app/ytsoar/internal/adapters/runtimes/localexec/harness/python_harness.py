# Runs inside a fresh `python3 -I` child spawned by the Go worker.
# Payload arrives as JSON on stdin, the result leaves as JSON on stdout.
# Parity target: connectors/code_snippet/operation.py python_inline —
# exec() the code, read the `result` variable, emit {"code_output": result}.
# Templating parity: connectors/core/connector.py evaluate_params renders
# every string with jinja2 as Template(value).render(var=variables).
import json
import sys


def render(value, variables):
    if isinstance(value, dict):
        return {key: render(val, variables) for key, val in value.items()}
    if isinstance(value, list):
        return [render(item, variables) for item in value]
    if isinstance(value, str) and ("{{" in value or "{%" in value):
        # lazy import: plain snippets run even without jinja2 installed
        from jinja2 import Template

        return Template(value).render(var=variables)
    return value


def main():
    payload = json.load(sys.stdin)
    variables = {"steps": payload.get("steps") or {}}
    params = render(payload.get("params") or {}, variables)
    code = params.get("code") or ""

    # stdout is the JSON result channel: reroute user print()s to stderr so
    # they cannot corrupt it.
    result_out = sys.stdout
    sys.stdout = sys.stderr

    # One namespace for globals AND locals: with separate dicts, module-level
    # names bind into locals but function bodies resolve against globals, so any
    # helper/recursive function or class referencing a top-level name raises
    # NameError. A single dict makes user code behave like a normal module.
    namespace = {"params": params, "steps": variables["steps"]}
    exec(compile(code, "user_code", "exec"), namespace)
    print(json.dumps({"code_output": namespace.get("result")}, default=str), file=result_out)


main()
