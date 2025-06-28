"""
workflow core file
"""

from celery import group, chain
from collections import deque
from collections.abc import Callable
from typing import Dict
from src.workers.celery import task_graph, workflow_completed


class WorkflowGraph:
    """
    class responsible for generating chain tasks
    """

    def __init__(
        self,
        graph: Dict[str, list[str]],
        task_information: dict,
        workflow_history_id: str,
    ):
        self.graph = graph
        self.task_information = task_information
        self.workflow_history_id = workflow_history_id

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
