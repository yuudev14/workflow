"""
workflow core file
"""
from graphlib import TopologicalSorter
from concurrent.futures import ThreadPoolExecutor, as_completed
from celery import group, chain
from collections import deque
from collections.abc import Callable
from connectors.core.connector import Connector
from typing import Dict, TypedDict, Optional, Any
from kombu import Connection, Queue
import traceback
from src.workers.celery import task_graph, workflow_completed
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
            self.send_workflow_status(
                status="failed", error=str(e) + "\n" + error_trace
            )
            raise e

    
    def execute_task(
        self,
    ):
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

    def generate_task(self, stepname: str):
        """
        Generate a Celery task signature for a given value.

        This function creates a Celery task signature using the `task_graph` task and a dictionary
        containing the given value. The task signature is returned for further use in task chaining.

        Args:
            val (str): The value to be passed to the task.

        Returns:
            A Celery task signature for the given value.
        """
        return task_graph.s(
            {
                stepname: None,
            },
            curr=stepname,
            task_information=self.task_information,
            workflow_history_id=self.workflow_history_id,
        )

    def generate_list_of_task(self, vals: list[str]):
        """
        Generate a list of Celery task signatures for a given list of values.

        This function creates a list of Celery task signatures using the `generate_task` method.
        If the input list contains only one value, a single task signature is returned.
        Otherwise, a Celery group of task signatures is returned.

        Args:
            vals (list[str]): A list of values to be passed to the tasks.

        Returns:
            celery.canvas.Group: A Celery group of task signatures for the given values.
            If the input list contains only one value, a single task signature is returned.
        """
        if len(vals) == 1:
            return self.generate_task(vals[0])
        return group([self.generate_task(val) for val in vals])
    

    def generate_chain_task(self):
        """
        Generates a chain of tasks based on the graph provided

        This function performs a breadth-first search (BFS) on the graph to generate a list of task

        The list of task signatures is then used to create a Celery chain using the `chain` function.

        Returns:
            A Celery chain of tasks based on the graph provided.
        """

        if self.is_acyclic_graph():
            raise Exception()

        task_chain_list = []

        self.bfs(lambda x: task_chain_list.append(self.generate_list_of_task(x)))

        task_chain = chain(
            *task_chain_list,
            workflow_completed.s(
                task_information=self.task_information,
                workflow_history_id=self.workflow_history_id,
            )
        )
        return task_chain.apply_async()

    def bfs(self, callback: Callable, node: str = "start"):
        visit = set()
        queue = deque()
        visit.add(node)
        queue.append(node)

        while queue:
            callback(list(queue))
            for _ in range(len(queue)):
                curr = queue.popleft()
                for neighbor in self.graph[curr]:
                    if neighbor not in visit:
                        visit.add(neighbor)
                        queue.append(neighbor)

    def generate_chain_task_using_topological_sort(self):
        """Generates a chain of tasks based on the graph provided."""
        if self.is_acyclic_graph():
            raise Exception()
        task_groups = self.topological_sort_with_groups()
        
        # Create a chain from the groups of tasks
        task_chain = chain(
            *task_groups,
            workflow_completed.s(
                task_information=self.task_information,
                workflow_history_id=self.workflow_history_id,
            )
        )
        return task_chain.apply_async()

    def topological_sort_with_groups(self) -> list[Callable]:
        """
        Perform topological sorting on the graph and create groups for parallel execution.
        
        """
        indegree = {u: 0 for u in self.graph}
        for u in self.graph:
            for v in self.graph[u]:
                indegree[v] += 1

        queue = deque()
        # Start with nodes that have zero indegree
        for u in indegree:
            if indegree[u] == 0:
                queue.append(u)

        sorted_task_groups = []

        while queue:
            current_group = [] 
            current_size = len(queue)

            for _ in range(current_size):
                node = queue.popleft()
                current_group.append(self.generate_task(node))
                
                # Process neighbors
                for neighbor in self.graph[node]:
                    indegree[neighbor] -= 1
                    if indegree[neighbor] == 0:
                        queue.append(neighbor)

            sorted_task_groups.append(group(*current_group)) 

        return sorted_task_groups


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
