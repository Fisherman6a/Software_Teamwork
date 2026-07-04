#
#  Copyright 2026 The InfiniFlow Authors. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
import importlib.util
import sys
import types
import warnings
from types import SimpleNamespace

import pytest

warnings.filterwarnings(
    "ignore",
    message="pkg_resources is deprecated as an API.*",
    category=UserWarning,
)


def _install_cv2_stub_if_unavailable():
    try:
        importlib.import_module("cv2")
        return
    except Exception:
        pass

    stub = types.ModuleType("cv2")
    stub.INTER_LINEAR = 1
    stub.INTER_CUBIC = 2
    stub.BORDER_CONSTANT = 0
    stub.BORDER_REPLICATE = 1

    def _missing(*_args, **_kwargs):
        raise RuntimeError("cv2 runtime call is unavailable in this test environment")

    def _module_getattr(name):
        if name.isupper():
            return 0
        return _missing

    stub.__getattr__ = _module_getattr
    sys.modules["cv2"] = stub


def _install_xgboost_stub_if_unavailable():
    if "xgboost" in sys.modules:
        return
    if importlib.util.find_spec("xgboost") is not None:
        return
    sys.modules["xgboost"] = types.ModuleType("xgboost")


_install_cv2_stub_if_unavailable()
_install_xgboost_stub_if_unavailable()

from api.db.services import file_service as file_service_module  # noqa: E402
from api.db.services.file_service import FileService  # noqa: E402


def _unwrapped_delete_docs():
    return FileService.delete_docs.__func__.__wrapped__


@pytest.mark.p2
def test_delete_docs_runs_document_cleanup_inside_document_schedule_locks(monkeypatch):
    events = []

    class FakeDocumentLock:
        def __init__(self, doc_ids):
            self.doc_ids = doc_ids

        def __enter__(self):
            events.append(("lock_enter", tuple(self.doc_ids)))
            return self

        def __exit__(self, exc_type, exc, tb):
            events.append(("lock_exit", tuple(self.doc_ids)))
            return False

    def fake_document_schedule_locks(doc_ids):
        events.append(("lock_create", tuple(doc_ids)))
        return FakeDocumentLock(doc_ids)

    docs = {
        "doc-b": SimpleNamespace(id="doc-b", kb_id="kb-1", parser_id="naive"),
        "doc-a": SimpleNamespace(id="doc-a", kb_id="kb-1", parser_id="naive"),
    }

    monkeypatch.setattr(FileService, "get_root_folder", classmethod(lambda cls, _scope_id: {"id": "root"}))
    monkeypatch.setattr(FileService, "init_knowledgebase_docs", classmethod(lambda cls, _pf_id, _scope_id: None))
    monkeypatch.setattr(file_service_module, "document_schedule_locks", fake_document_schedule_locks)
    monkeypatch.setattr(file_service_module.DocumentService, "get_by_id", lambda doc_id: (True, docs[doc_id]))
    monkeypatch.setattr(file_service_module.DocumentService, "get_scope_id", lambda _doc_id: "scope-1")
    monkeypatch.setattr(file_service_module.File2DocumentService, "get_storage_address", lambda doc_id: ("bucket", doc_id))
    monkeypatch.setattr(file_service_module.TaskService, "filter_delete", lambda _filters: events.append(("task_delete",)))
    monkeypatch.setattr(
        file_service_module.DocumentService,
        "remove_document",
        lambda doc, _scope_id: events.append(("remove_document", doc.id)) or True,
    )
    monkeypatch.setattr(file_service_module.File2DocumentService, "get_by_document_id", lambda _doc_id: [])
    monkeypatch.setattr(
        file_service_module.File2DocumentService,
        "delete_by_document_id",
        lambda doc_id: events.append(("unlink_file", doc_id)),
    )

    errors = _unwrapped_delete_docs()(FileService, ["doc-b", "doc-a"], "scope-1")

    assert errors == ""
    assert events == [
        ("lock_create", ("doc-b", "doc-a")),
        ("lock_enter", ("doc-b", "doc-a")),
        ("task_delete",),
        ("remove_document", "doc-b"),
        ("unlink_file", "doc-b"),
        ("task_delete",),
        ("remove_document", "doc-a"),
        ("unlink_file", "doc-a"),
        ("lock_exit", ("doc-b", "doc-a")),
    ]
