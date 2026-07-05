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


def main():
    payload = json.load(sys.stdin)
    root = payload["connectors_root"]
    os.chdir(root)
    sys.path.insert(0, root)

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
    print(json.dumps(result, default=str))


main()
