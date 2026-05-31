from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class AnalyzeRequestMessage(_message.Message):
    __slots__ = ("project_id", "client_ip", "method", "path", "headers", "body")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    CLIENT_IP_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    client_ip: str
    method: str
    path: str
    headers: _containers.ScalarMap[str, str]
    body: str
    def __init__(self, project_id: _Optional[str] = ..., client_ip: _Optional[str] = ..., method: _Optional[str] = ..., path: _Optional[str] = ..., headers: _Optional[_Mapping[str, str]] = ..., body: _Optional[str] = ...) -> None: ...

class AnalyzeResponseMessage(_message.Message):
    __slots__ = ("risk_score", "block_recommended", "reason", "flags")
    RISK_SCORE_FIELD_NUMBER: _ClassVar[int]
    BLOCK_RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FLAGS_FIELD_NUMBER: _ClassVar[int]
    risk_score: int
    block_recommended: bool
    reason: str
    flags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, risk_score: _Optional[int] = ..., block_recommended: bool = ..., reason: _Optional[str] = ..., flags: _Optional[_Iterable[str]] = ...) -> None: ...
