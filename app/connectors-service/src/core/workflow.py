"""
workflow core file
"""
from graphlib import TopologicalSorter
from concurrent.futures import ThreadPoolExecutor, as_completed
from connectors.core.connector import Connector
from typing import Dict, TypedDict, Optional, Any
from kombu import Connection, Queue
import traceback
from src.logger.logging import logger
from src.settings import settings
from src import dto




class Task(TypedDict):
    id: str
    workflow_id: str
    name: str
    description: Optional[str]
    parameters: Optional[Any]
    config: Optional[str]
    x: float
    y: float
    connector_name: Optional[str]
    connector_id: Optional[str]
    operation: Optional[str]


class Store(TypedDict):
    steps: dict


class WorkflowGraph:
    """
    class responsible for generating chain tasks
    """

    def __init__(
        self,
        graph: Dict[str, list[str]],
        task_information: Dict[str, Task],
        workflow_history_id: str,
    ):
        self.graph = graph
        self.task_information = task_information
        self.workflow_history_id = workflow_history_id
        self.skip_nodes = set()
        self.store: Store = {
            "steps": {}
        }
        self.queue = Queue(settings.workflow_processor_queue, durable=True)

    def invert_graph(self, successor_graph: dict):
        """
        invert graph from node values to it's direction to node values to it's dependency
        """
        inverted = {}

        # grab all the nodes
        all_nodes = set(successor_graph.keys())
        for val in successor_graph.values():
            all_nodes.update(val)

        for node in all_nodes:
            inverted[node] = set()

        # add the dependency in the map
        for node, targets in successor_graph.items():
            for target in targets:
                inverted[target].add(node)

        return inverted
    
    def _send_message_to_mq(self, message: Any):
        with Connection(settings.mq_url) as conn:

            # Produce a message
            with conn.Producer() as producer:
                producer.publish(
                    message,
                    exchange="",
                    routing_key=self.queue.name,
                    declare=[self.queue],
                    delivery_mode=2,  # Make message persistent
                )


    def send_workflow_status(
        self,
        status: dto.message_payload.WorkflowStatus,
        result: Any | None = None,
        error: str | None = None,
    ):
        payload = dto.message_payload.MessageProcessorPayload(
            action="workflow_status",
            params=dto.message_payload.WorkflowStatusPayload(
                workflow_history_id=self.workflow_history_id,
                status=status,
                result=result,
                error=error,
            ),
        )
        return self._send_message_to_mq(payload.model_dump_json())


    def send_task_status(
        self,
        task_id: str,
        task: Task,
        status: dto.message_payload.TaskStatus,
        result: Any | None = None,
        error: str | None = None,
    ):
        
        needed_fields = [
            "name", "description", "parameters", "connector_name",
            "connector_id", "operation", "config", "x", "y"
        ]

        extracted_task_fields = {key: task[key] for key in needed_fields}
        payload = dto.message_payload.MessageProcessorPayload(
            action="task_status",
            params=dto.message_payload.TaskStatusPayload(
                workflow_history_id=self.workflow_history_id,
                task_id=task_id,
                status=status,
                result=result,
                error=error,
                **extracted_task_fields,
            ),
        )
        return self._send_message_to_mq(payload.model_dump_json())
    
    def process_node(
        self,
        node,
    ):
        logger.info(f"executing {node} in playbook.")

        if node not in self.task_information:
            raise Exception(f"operation ({node}) does not exist in task_information")
        if node is None:
            logger.warning("'curr' is not available in kwargs")
            return
        
        task_information = self.task_information[node]
        config_name = task_information.get("config", None)
        parameters = task_information.get("parameters", None)
        connector_id = task_information.get("connector_id", None)
        operation = task_information.get("operation", None)

        try:

            self.send_task_status(
                task_id=task_information.get("id"),
                status="in_progress",
                task=task_information,
            )


            if connector_id is None and node != "start":
                raise Exception(f"connector id is none for {node}")

            if node == "start":
                self.send_task_status(
                    task_id=task_information.get("id"),
                    status="success",
                    task=task_information,
                )

                return None

            # get the class container
            connector = Connector.get_class_container(connector_id)

            # grab the config to use
            config = Connector.get_connector_config(
                config_name=config_name, connector_id=connector_id
            )

            params = Connector.evaluate_params(
                parameters=parameters, variables=self.store
            )

            # execute the operations
            operation_result = connector.execute(
                configs=config, params=params, operation=operation
            )

            # assign the result for the operation
            logger.debug(f"execution complete for playbook, {operation=}")
            self.store["steps"][node] = operation_result
            self.send_task_status(
                task_id=task_information.get("id"),
                status="success",
                result=self.store["steps"][node],
                task=task_information,
            )

            return {
                node: operation_result
            }

        except Exception as e:
            #  send log
            error_trace = traceback.format_exc()
            self.send_task_status(
                task_id=task_information.get("id"),
                status="failed",
                error=str(e) + "\n" + error_trace,
                task=task_information,
            )
            raise e

    
    def execute_task(
        self,
    ):
        try:
            predecessor_graph = self.invert_graph(self.graph)
            ts = TopologicalSorter(predecessor_graph)
            ts.prepare()

            with ThreadPoolExecutor() as executor:
                while ts.is_active():
                    ready_nodes = ts.get_ready()

                    for node in ready_nodes:
                        if node in self.skip_nodes:
                            ts.done(node)
                            continue

                    nodes_to_execute = [node for node in ready_nodes if node not in self.skip_nodes]
                    logger.info(f"processing {nodes_to_execute}")
                    futures = {
                        executor.submit(self.process_node, node): node
                        for node in nodes_to_execute
                    }

                    for future in as_completed(futures):
                        node_done = futures[future]
                        result = future.result()
                        logger.info(f"result of the node {result=}")
                        ts.done(node_done)
            
            self.send_workflow_status("success", result=self.store["steps"])
            return self.store
        except Exception as e:
            error_trace = traceback.format_exc()
            self.send_workflow_status(
                status="failed", error=str(e) + "\n" + error_trace
            )
            raise e

    def is_acyclic_graph(self):
        visit: set = set()
        stack: set = set()

        def dfs(node: str):
            if node in stack:
                return True

            if node in visit:
                return False

            visit.add(node)
            stack.add(node)

            for neighbor in self.graph[node]:
                if dfs(neighbor):
                    return True

            stack.remove(node)
            return False

        for node in self.graph:
            if dfs(node):
                return True

        return False


# tests
if __name__ == "__main__":
    task_information = {
        "START": {},
        "A": {
            "config": "sample",
        },
        "B": {
            "config": "sample",
        },
        "C": {
            "config": "sample",
        },
        "D": {
            "config": "sample",
        },
        "E": {
            "config": "sample",
        },
        "F": {
            "config": "sample",
        },
    }
    graph = {
        "START": ["A"],
        "A": ["B", "C"],
        "B": ["D"],
        "C": ["E", "D"],
        "D": [],
        "E": ["F"],
        "F": [],
    }

    x = WorkflowGraph(graph=graph, task_information=task_information)
