from fastapi import APIRouter, HTTPException
from src.core.workflow import WorkflowGraph
from src.logger.logging import logger
import logging


router = APIRouter()


test_data = {
    "graph": {
        "name": [],
        "name2": ["name3"],
        "name3": ["name"],
        "start": ["name", "name2"],
    },
    "tasks": {
        "name": {
            "id": "4014a8bc-9614-43e4-953b-a8702c8f28fd",
            "workflow_id": "87d8dcfd-c56b-4d63-b1e1-6cb32495ff96",
            "status": "",
            "name": "name",
            "config": "",
            "connector_name": "Code Snippet",
            "connector_id": "code_snippet",
            "operation": "python_code",
            "description": "",
            "parameters": {
                "code": 'def main():\r\n    return [{{var.steps["name2"]}}, {{var.steps["name3"]}}]\r\n\r\n\r\nresult = main()'
            },
            "created_at": "2025-07-26T03:14:45.600914Z",
            "updated_at": "2025-07-26T03:56:24.404855Z",
            "position": {},
            "x": 982,
            "y": 189,
        },
        "name2": {
            "id": "47a8b47d-26f0-4ad9-a357-d85abe473369",
            "workflow_id": "87d8dcfd-c56b-4d63-b1e1-6cb32495ff96",
            "status": "",
            "name": "name2",
            "config": "",
            "connector_name": "Code Snippet",
            "connector_id": "code_snippet",
            "operation": "python_code",
            "description": "",
            "parameters": {
                "code": "def main():\r\n    return 2\r\n\r\n\r\nresult = main()"
            },
            "created_at": "2025-07-26T03:51:40.51754Z",
            "updated_at": "2025-07-26T03:56:24.404855Z",
            "position": {},
            "x": 131,
            "y": 439,
        },
        "name3": {
            "id": "ba08cd0d-ab83-4e28-9df9-2ba7d7e40261",
            "workflow_id": "87d8dcfd-c56b-4d63-b1e1-6cb32495ff96",
            "status": "",
            "name": "name3",
            "config": "",
            "connector_name": "Code Snippet",
            "connector_id": "code_snippet",
            "operation": "python_code",
            "description": "",
            "parameters": {
                "code": "def main():\r\n    return 3\r\n\r\n\r\nresult = main()"
            },
            "created_at": "2025-07-26T03:51:40.51754Z",
            "updated_at": "2025-07-26T03:56:24.404855Z",
            "position": {},
            "x": 481,
            "y": 673,
        },
        "start": {
            "id": "3ebded4c-7ae2-4ca3-b53f-54b9905a99e6",
            "workflow_id": "87d8dcfd-c56b-4d63-b1e1-6cb32495ff96",
            "status": "",
            "name": "start",
            "config": None,
            "connector_name": None,
            "connector_id": None,
            "operation": "",
            "description": "",
            "parameters": None,
            "created_at": "2025-07-21T13:32:56.366424Z",
            "updated_at": "2025-07-26T03:56:24.404855Z",
            "position": {},
            "x": 100,
            "y": 100,
        },
    },
    "workflow_history_id": "290ed766-78cd-4708-8943-bc4bde96931d",
}




class CeleryController:
    def __init__(self):
        pass

    @router.get("/celery")
    async def celery_workflow():
        data = test_data
        task_information = data["tasks"]
        graph = data["graph"]
        logger.debug(f"{graph=}")

        x = WorkflowGraph(
            graph=graph, task_information=task_information, workflow_history_id="1be1c8bd-c5f4-4469-8d34-6b4563238326"
        )
        return x.execute_task()
