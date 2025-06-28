from connectors.core.connector import Connector
import time


class Sample(Connector):

    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        print(f"executed, {operation} {params}")
        return {"sample": f"executed, {operation} {params}"}

    def health_check(
        self, configs: dict, params: dict, operation: str, *args, **kwargs
    ):
        print(f"executed, {operation}")
        return operation
