from celery import Celery

from src.logger.logging import logger
from src.core.workflow import WorkflowGraph
from src.settings import settings
from src import dto

celery_app = Celery(
    "tasks", broker=settings.celery_broker, backend=settings.celery_backend
)


@celery_app.task
def task_graph(*args: tuple[dict] | dict | list[dict], **kwargs):
    task_information = kwargs["task_information"]
    graph = kwargs["graph"]
    workflow_history_id = kwargs["workflow_history_id"]
    logger.debug(f"{graph=}")

    x = WorkflowGraph(
        graph=graph,
        task_information=task_information,
        workflow_history_id=workflow_history_id,
    )
    return x.execute_task()


# @task_prerun.connect(sender=task_graph)
# def task_prerun_handler(sender=None, **kwargs):
#     print("Pre-execution: Task is about to run")


# @task_success.connect(sender=task_graph)
# def task_success_handler(sender=None, **kwargs):
#     print("Task completed successfully")


# @task_failure.connect(sender=task_graph)
# def task_failure_handler(sender=None, **kwargs):
#     print("Task failed")
