import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TaskStatusPayload(_message.Message):
    __slots__ = ("workflow_history_id", "task_id", "name", "description", "parameters", "connector_name", "connector_id", "operation", "config", "x", "y", "status", "error", "result")
    WORKFLOW_HISTORY_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    CONNECTOR_NAME_FIELD_NUMBER: _ClassVar[int]
    CONNECTOR_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    workflow_history_id: str
    task_id: str
    name: str
    description: str
    parameters: str
    connector_name: str
    connector_id: str
    operation: str
    config: str
    x: float
    y: float
    status: str
    error: str
    result: str
    def __init__(self, workflow_history_id: _Optional[str] = ..., task_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., parameters: _Optional[str] = ..., connector_name: _Optional[str] = ..., connector_id: _Optional[str] = ..., operation: _Optional[str] = ..., config: _Optional[str] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., result: _Optional[str] = ...) -> None: ...

class WorkflowStatusPayload(_message.Message):
    __slots__ = ("workflow_history_id", "status", "error", "result")
    WORKFLOW_HISTORY_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    workflow_history_id: str
    status: str
    error: str
    result: str
    def __init__(self, workflow_history_id: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., result: _Optional[str] = ...) -> None: ...

class WorkflowHistory(_message.Message):
    __slots__ = ("id", "workflow_id", "status", "error", "result", "triggered_at", "edges")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    TRIGGERED_AT_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    id: str
    workflow_id: str
    status: str
    error: str
    result: _struct_pb2.Value
    triggered_at: _timestamp_pb2.Timestamp
    edges: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., result: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., triggered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., edges: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class TaskHistory(_message.Message):
    __slots__ = ("id", "workflow_history_id", "task_id", "status", "error", "result", "triggered_at", "name", "config", "connector_name", "connect_id", "operation", "description", "parameters", "x", "y", "destination_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_HISTORY_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    TRIGGERED_AT_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    CONNECTOR_NAME_FIELD_NUMBER: _ClassVar[int]
    CONNECT_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    workflow_history_id: str
    task_id: str
    status: str
    error: str
    result: _struct_pb2.Value
    triggered_at: _timestamp_pb2.Timestamp
    name: str
    config: str
    connector_name: str
    connect_id: str
    operation: str
    description: str
    parameters: _struct_pb2.Value
    x: float
    y: float
    destination_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., workflow_history_id: _Optional[str] = ..., task_id: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., result: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., triggered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., name: _Optional[str] = ..., config: _Optional[str] = ..., connector_name: _Optional[str] = ..., connect_id: _Optional[str] = ..., operation: _Optional[str] = ..., description: _Optional[str] = ..., parameters: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., destination_ids: _Optional[_Iterable[str]] = ...) -> None: ...
