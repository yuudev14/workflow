"""
gRPC ConnectorRuntime server.

Remote replacement for the in-process ``connector.execute()`` call: the Go
executor sends raw task parameters plus the accumulated steps snapshot, and
templating (jinja2) happens here so connector semantics stay identical to the
Python executor path.
"""

import json
import traceback
from concurrent import futures

import grpc

from connectors.core.connector import Connector
from src.grpc.connector_runtime import connector_runtime_pb2 as pb
from src.grpc.connector_runtime import connector_runtime_pb2_grpc as pb_grpc
from src.logger.logging import logger


class ConnectorRuntimeService(pb_grpc.ConnectorRuntimeServicer):
    def ExecuteOperation(self, request, context):
        try:
            steps = json.loads(request.steps_json) if request.steps_json else {}
            parameters = (
                json.loads(request.parameters_json) if request.parameters_json else None
            )

            connector = Connector.get_class_container(request.connector_id)
            config = Connector.get_connector_config(
                config_name=request.config_name if request.HasField("config_name") else None,
                connector_id=request.connector_id,
            )
            # variables shape matches the executor store: {{ var.steps["node"] }}
            params = Connector.evaluate_params(
                parameters=parameters, variables={"steps": steps}
            )
            result = connector.execute(
                configs=config, params=params, operation=request.operation
            )
            return pb.ExecuteOperationResponse(
                result_json=json.dumps(result, default=str)
            )
        except Exception as e:
            logger.error(f"ExecuteOperation failed for {request.connector_id}: {e}")
            return pb.ExecuteOperationResponse(
                error=f"{e}\n{traceback.format_exc()}"
            )

    def HealthCheck(self, request, context):
        return pb.HealthCheckResponse(ok=True)


def serve(port: int) -> grpc.Server:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pb_grpc.add_ConnectorRuntimeServicer_to_server(ConnectorRuntimeService(), server)
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    logger.info(f"ConnectorRuntime gRPC server listening on :{port}")
    return server
