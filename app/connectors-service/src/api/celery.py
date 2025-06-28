from fastapi import APIRouter, HTTPException
from src.core.workflow import WorkflowGraph
from src.logger.logging import logger
import logging


router = APIRouter()


class CeleryController:
    def __init__(self):
        pass

    @router.get("/celery")
    async def celery_workflow():

        data = {
            "graph": {
                "2": ["4"],
                "3": ["4"],
                "4": [],
                "create authentication": ["2", "3"],
                "start": ["create authentication"],
            },
            "tasks": {
                "2": {
                    "id": "59af74ed-f625-4907-912e-3163661271bb",
                    "workflow_id": "52e2016e-3cb2-410b-9fb6-f0f4a23785ff",
                    "status": "",
                    "name": "2",
                    "config": None,
                    "connector_name": "sample",
                    "operation": "sample",
                    "description": "aasdasds",
                    "parameters": {"x": 1},
                    "created_at": "2025-03-09T07:27:21.477727Z",
                    "updated_at": "2025-03-09T08:19:16.644489Z",
                    "position": {},
                    "x": 250,
                    "y": 350,
                },
                "3": {
                    "id": "a5e782bd-c2c2-4b07-98be-9a600bcee56d",
                    "workflow_id": "52e2016e-3cb2-410b-9fb6-f0f4a23785ff",
                    "status": "",
                    "name": "3",
                    "config": None,
                    "connector_name": "sample",
                    "operation": "sample",
                    "description": "asdasdas",
                    "parameters": {},
                    "created_at": "2025-03-09T07:27:21.477727Z",
                    "updated_at": "2025-03-09T08:19:16.644489Z",
                    "position": {},
                    "x": 60,
                    "y": 300,
                },
                "4": {
                    "id": "1a076be9-4c12-461f-ab95-699fe42f8715",
                    "workflow_id": "52e2016e-3cb2-410b-9fb6-f0f4a23785ff",
                    "status": "",
                    "name": "4",
                    "config": None,
                    "connector_name": "sample",
                    "operation": "sample",
                    "description": "1",
                    "parameters": {},
                    "created_at": "2025-03-09T07:27:21.477727Z",
                    "updated_at": "2025-03-09T08:19:16.644489Z",
                    "position": {},
                    "x": 220,
                    "y": 120,
                },
                "create authentication": {
                    "id": "e43f438c-6a15-45de-8ed5-838006383b8c",
                    "workflow_id": "52e2016e-3cb2-410b-9fb6-f0f4a23785ff",
                    "status": "",
                    "name": "create authentication",
                    "config": None,
                    "connector_name": "sample",
                    "operation": "sample",
                    "description": "As you can see we added two source handles to the node so that it has two outputs. If you want to connect other nodes with these specific handles, the node id is not enough but you also need to pass the specific handle id. In this case one handle has the id 'a' and the other one 'b'. Handle specific edges use the sourceHandle or targetHandle options that reference a handle within a node",
                    "parameters": None,
                    "created_at": "2025-03-09T07:27:21.477727Z",
                    "updated_at": "2025-03-09T08:19:16.644489Z",
                    "position": {},
                    "x": 500,
                    "y": 30,
                },
                "start": {
                    "id": "1311ddb5-f9eb-4511-8994-aa8a29fdad59",
                    "workflow_id": "52e2016e-3cb2-410b-9fb6-f0f4a23785ff",
                    "status": "",
                    "name": "start",
                    "config": "sample",
                    "connector_name": "sample",
                    "operation": "sample",
                    "description": "aasdasds",
                    "parameters": None,
                    "created_at": "2025-03-09T07:27:21.477727Z",
                    "updated_at": "2025-03-09T08:19:16.644489Z",
                    "position": {},
                    "x": 0,
                    "y": 0,
                },
            },
        }
        task_information = data["tasks"]
        graph = data["graph"]
        logger.debug(f"{graph=}")

        x = WorkflowGraph(
            graph=graph, task_information=task_information, workflow_history_id="1be1c8bd-c5f4-4469-8d34-6b4563238326"
        )
        x.generate_chain_task_using_topological_sort()
