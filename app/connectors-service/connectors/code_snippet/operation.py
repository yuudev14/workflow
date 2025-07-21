
def python_inline(configs: dict, params: dict, *args, **kwargs):
    # NOTE: need more check regarding avoiding imports
    code = params.get("code")
    code = compile(code, "code", "exec")
    local_vars = {}
    exec(code, {}, local_vars)
    return {
        "code_output": local_vars.get("result")
    }
operations = {
    "python_code": python_inline
}