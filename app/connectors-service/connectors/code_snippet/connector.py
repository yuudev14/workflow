from connectors.core.connector import Connector
from connectors.code_snippet.operation import operations


class Sample(Connector):

    def execute(self, configs: dict, params: dict, operation: str, *args, **kwargs):
        return operations[operation](configs, params)

    def health_check(
        self, configs: dict, params: dict, operation: str, *args, **kwargs
    ):
        print(f"executed, {operation}")
        return operation
