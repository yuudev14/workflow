# Runs inside a fresh `python3 -I` child spawned by the sandbox to execute one
# Python connector operation. Payload arrives as JSON on stdin, the result
# leaves as JSON on stdout.
#
# Parity is exact because this reuses connectors/core/connector.py verbatim:
# payload["connectors_root"] is the directory CONTAINING the connectors/
# package (and src/, which the core logger imports). get_connector_config
# opens ./connectors/<id>/configs/<name>.toml relative to the cwd, hence the
# chdir.
import json
import os
import sys


def apply_memory_limit():
    # Cap the address space (node gets --max-old-space-size; python has no
    # flag, so the harness applies RLIMIT_AS itself) so a runaway connector
    # cannot OOM the whole sandbox. Best-effort: never break the run.
    try:
        mb = int(os.environ.get("YTSOAR_MEM_LIMIT_MB", "0"))
        if mb > 0:
            import resource

            limit = mb * 1024 * 1024
            resource.setrlimit(resource.RLIMIT_AS, (limit, limit))
    except Exception:
        pass


def main():
    apply_memory_limit()
    payload = json.load(sys.stdin)
    root = payload["connectors_root"]
    os.chdir(root)
    sys.path.insert(0, root)

    # per-connector vendored dependencies: `make connector-deps` pip-installs
    # <id>/requirements.txt into <id>/deps with --target. Prepending keeps each
    # connector's pins isolated (fresh subprocess per run) and lets them win
    # over the image's baseline packages.
    deps = os.path.join(root, "connectors", payload["connector_id"], "deps")
    if os.path.isdir(deps):
        sys.path.insert(0, deps)

    # stdout is the JSON result channel: reroute connector prints/logs to
    # stderr so they cannot corrupt it.
    result_out = sys.stdout
    sys.stdout = sys.stderr

    from connectors.core.connector import Connector

    connector = Connector.get_class_container(payload["connector_id"])
    config = Connector.get_connector_config(
        config_name=payload.get("config_name"),
        connector_id=payload["connector_id"],
    )
    params = Connector.evaluate_params(
        parameters=payload.get("params"),
        variables={"steps": payload.get("steps") or {}},
    )
    result = connector.execute(
        configs=config, params=params, operation=payload["operation"]
    )
    print(json.dumps(result, default=str), file=result_out)


main()
