from pydantic import BaseModel
from typing import Literal, Any

# Define constants for status types
TaskStatus = Literal["pending", "success", "in_progress", "failed"]
WorkflowStatus = Literal["success", "in_progress", "failed"]


class TaskStatusPayload(BaseModel):
    workflow_history_id: str
    task_id: str
    status: TaskStatus
    result: Any | None = None
    error: str | None = None
    name: str | None = None
    description: str | None = None
    parameters: str | dict | None = None
    connector_name: str | None = None
    connector_id: str | None = None
    operation: str | None = None
    config: str | None = None
    x: float | None = None
    y: float | None = None

class WorkflowStatusPayload(BaseModel):
    workflow_history_id: str
    status: WorkflowStatus
    result: Any | None = None
    error: str | None = None

class MessageProcessorPayload(BaseModel):
    action: Literal["workflow_status", "task_status"]
    params: TaskStatusPayload | WorkflowStatusPayload