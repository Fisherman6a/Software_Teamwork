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
import logging
import os
import re
from typing import Any

from quart import request

from api.apps import login_required
from api.constants import FILE_NAME_LEN_LIMIT
from api.db import FileType
from api.utils.api_utils import get_error_argument_result, get_json_result, server_error_response
from api.utils.file_utils import filename_type
from api.utils.runtime_scope import runtime_scope_id
from common import settings
from common.constants import RetCode
from common.misc_utils import thread_pool_exec
from rag.app import naive, picture, presentation


DEFAULT_ATTACHMENT_PARSE_MAX_BYTES = 20 * 1024 * 1024
MAX_ATTACHMENT_PARSE_MAX_BYTES = 100 * 1024 * 1024


def _attachment_parse_max_bytes() -> int:
    raw = os.getenv("KNOWLEDGE_RUNTIME_ATTACHMENT_PARSE_MAX_BYTES", "").strip()
    if not raw:
        return DEFAULT_ATTACHMENT_PARSE_MAX_BYTES
    try:
        parsed = int(raw)
    except ValueError:
        return DEFAULT_ATTACHMENT_PARSE_MAX_BYTES
    if parsed <= 0:
        return DEFAULT_ATTACHMENT_PARSE_MAX_BYTES
    return min(parsed, MAX_ATTACHMENT_PARSE_MAX_BYTES)


def _safe_filename(value: str) -> str:
    name = os.path.basename(str(value or "")).strip()
    if not name or len(name) > FILE_NAME_LEN_LIMIT:
        return ""
    return name


def _default_parser_config() -> dict[str, Any]:
    return {
        "chunk_token_num": 512,
        "delimiter": "\n!?;。；！？",
        "layout_recognize": "DeepDOC",
        "table_context_size": 0,
        "image_context_size": 0,
        "analyze_hyperlink": False,
    }


def _parser_for_filename(filename: str):
    lower = filename.lower()
    if re.search(r"\.pptx?$", lower):
        return presentation
    if filename_type(filename) == FileType.VISUAL.value:
        return picture
    return naive


def _chunk_text(chunk: dict[str, Any]) -> str:
    for key in ("content_with_weight", "content", "text", "q", "a"):
        value = chunk.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""


def _chunk_page_number(chunk: dict[str, Any]) -> int:
    for key in ("page_num_int", "page_number", "page_num"):
        value = chunk.get(key)
        if isinstance(value, int) and value > 0:
            return value
        if isinstance(value, str) and value.strip().isdigit():
            parsed = int(value.strip())
            if parsed > 0:
                return parsed
    return 1


def _chunk_section_path(chunk: dict[str, Any]) -> str:
    for key in ("section_path", "section", "title", "docnm_kwd"):
        value = chunk.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""


def _parse_attachment_sync(filename: str, binary: bytes, language: str) -> dict[str, Any]:
    parser_config = _default_parser_config()
    chunker = _parser_for_filename(filename)
    chunks = chunker.chunk(
        filename,
        binary=binary,
        lang=language or "Chinese",
        callback=lambda *args, **kwargs: None,
        parser_config=parser_config,
        scope_id=runtime_scope_id(),
        kb_id="qa_session_attachment",
        llm_id="",
    )

    parsed_chunks = []
    max_page = 1
    for item in chunks or []:
        if not isinstance(item, dict):
            continue
        content = _chunk_text(item)
        if not content:
            continue
        page_number = _chunk_page_number(item)
        max_page = max(max_page, page_number)
        parsed_chunks.append(
            {
                "pageNumber": page_number,
                "sectionPath": _chunk_section_path(item),
                "content": content,
            }
        )

    return {
        "pageCount": max_page,
        "chunks": parsed_chunks,
    }


@manager.route("/internal/attachments/parse", methods=["POST"])  # noqa: F821
@login_required
async def parse_attachment():
    files = await request.files
    file_obj = files.get("file") if files else None
    if file_obj is None:
        return get_error_argument_result("file is required")

    filename = _safe_filename(file_obj.filename)
    if not filename:
        return get_error_argument_result("filename is required")

    filetype = filename_type(filename)
    if filetype == FileType.OTHER.value:
        return get_error_argument_result("unsupported file type")

    binary = file_obj.read()
    if not binary:
        return get_error_argument_result("file is empty")
    max_bytes = _attachment_parse_max_bytes()
    if len(binary) > max_bytes:
        return get_json_result(code=RetCode.ARGUMENT_ERROR, message="file exceeds attachment parse limit"), 413

    form = await request.form
    language = (form.get("language") or "Chinese").strip() or "Chinese"
    try:
        data = await thread_pool_exec(_parse_attachment_sync, filename, binary, language)
    except Exception as exc:
        logging.exception("temporary attachment parse failed for %s", re.sub(r"[\r\n]+", " ", filename))
        return server_error_response(exc)

    return get_json_result(data=data)
