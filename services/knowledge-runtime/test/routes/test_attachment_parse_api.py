import io
import sys

import pytest
from werkzeug.datastructures import FileStorage

from api.apps import app
from api.utils.gateway_auth import SERVICE_TOKEN_HEADER


def _file_storage(content: bytes, filename: str, content_type: str = "application/octet-stream") -> FileStorage:
    return FileStorage(stream=io.BytesIO(content), filename=filename, content_type=content_type, name="file")


@pytest.mark.asyncio
async def test_attachment_parse_requires_runtime_service_token(monkeypatch):
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_SERVICE_TOKEN", "runtime-token")

    client = app.test_client()
    response = await client.post(
        "/api/v1/internal/attachments/parse",
        files={"file": _file_storage(b"hello", "note.txt", "text/plain")},
    )

    assert response.status_code == 401
    payload = await response.get_json()
    assert payload["code"] == 401


@pytest.mark.asyncio
async def test_attachment_parse_maps_runtime_chunks(monkeypatch):
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_SERVICE_TOKEN", "runtime-token")

    attachment_parse_api = sys.modules["api.apps.restful_apis.attachment_parse_api"]

    calls = {}

    def fake_chunk(filename, binary=None, **kwargs):
        calls["filename"] = filename
        calls["binary"] = binary
        calls["parser_config"] = kwargs.get("parser_config")
        return [
            {"content_with_weight": " first chunk ", "page_num_int": 2, "section_path": "intro"},
            {"content": "second chunk", "page_number": "3"},
            {"content": "   "},
        ]

    monkeypatch.setattr(attachment_parse_api.naive, "chunk", fake_chunk)

    client = app.test_client()
    response = await client.post(
        "/api/v1/internal/attachments/parse",
        headers={SERVICE_TOKEN_HEADER: "runtime-token"},
        form={"language": "Chinese"},
        files={"file": _file_storage(b"%PDF", "manual.pdf", "application/pdf")},
    )

    assert response.status_code == 200
    payload = await response.get_json()
    assert payload["code"] == 0
    assert payload["data"] == {
        "pageCount": 3,
        "chunks": [
            {"pageNumber": 2, "sectionPath": "intro", "content": "first chunk"},
            {"pageNumber": 3, "sectionPath": "", "content": "second chunk"},
        ],
    }
    assert calls["filename"] == "manual.pdf"
    assert calls["binary"] == b"%PDF"
    assert calls["parser_config"]["analyze_hyperlink"] is False


def test_attachment_parse_selects_runtime_parser_by_file_type():
    attachment_parse_api = sys.modules["api.apps.restful_apis.attachment_parse_api"]

    assert attachment_parse_api._parser_for_filename("slides.pptx") is attachment_parse_api.presentation
    assert attachment_parse_api._parser_for_filename("image.png") is attachment_parse_api.picture
    assert attachment_parse_api._parser_for_filename("manual.pdf") is attachment_parse_api.naive


@pytest.mark.asyncio
async def test_attachment_parse_rejects_unsupported_file_type(monkeypatch):
    monkeypatch.setenv("KNOWLEDGE_RUNTIME_SERVICE_TOKEN", "runtime-token")

    client = app.test_client()
    response = await client.post(
        "/api/v1/internal/attachments/parse",
        headers={SERVICE_TOKEN_HEADER: "runtime-token"},
        files={"file": _file_storage(b"hello", "archive.bin")},
    )

    assert response.status_code == 200
    payload = await response.get_json()
    assert payload["code"] != 0
    assert "unsupported file type" in payload["message"]
