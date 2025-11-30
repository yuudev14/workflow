import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
