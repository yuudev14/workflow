from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExecuteOperationRequest(_message.Message):
    __slots__ = ("connector_id", "operation", "config_name", "parameters_json", "steps_json", "playbook_history_id", "task_id", "timeout_ms")
    CONNECTOR_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CONFIG_NAME_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_JSON_FIELD_NUMBER: _ClassVar[int]
    STEPS_JSON_FIELD_NUMBER: _ClassVar[int]
    PLAYBOOK_HISTORY_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    connector_id: str
    operation: str
    config_name: str
    parameters_json: str
    steps_json: str
    playbook_history_id: str
    task_id: str
    timeout_ms: int
    def __init__(self, connector_id: _Optional[str] = ..., operation: _Optional[str] = ..., config_name: _Optional[str] = ..., parameters_json: _Optional[str] = ..., steps_json: _Optional[str] = ..., playbook_history_id: _Optional[str] = ..., task_id: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ExecuteOperationResponse(_message.Message):
    __slots__ = ("result_json", "error")
    RESULT_JSON_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    result_json: str
    error: str
    def __init__(self, result_json: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class HealthCheckRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthCheckResponse(_message.Message):
    __slots__ = ("ok",)
    OK_FIELD_NUMBER: _ClassVar[int]
    ok: bool
    def __init__(self, ok: _Optional[bool] = ...) -> None: ...
