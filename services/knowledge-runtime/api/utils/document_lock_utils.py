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


logger = logging.getLogger(__name__)

DOCUMENT_SCHEDULE_LOCK_PREFIX = "document_schedule"
DOCUMENT_SCHEDULE_LOCK_BUSY_FRAGMENT = "busy with another scheduling operation"
DOCUMENT_SCHEDULE_LOCK_TIMEOUT_ENV = "KNOWLEDGE_RUNTIME_DOCUMENT_SCHEDULE_LOCK_TIMEOUT_SECONDS"
DOCUMENT_SCHEDULE_LOCK_BLOCKING_TIMEOUT_ENV = "KNOWLEDGE_RUNTIME_DOCUMENT_SCHEDULE_LOCK_BLOCKING_TIMEOUT_SECONDS"


def _positive_int_env(name: str, default: int) -> int:
    value = os.environ.get(name)
    if value is None:
        return default
    try:
        parsed = int(value)
    except ValueError:
        logger.warning("Ignoring invalid integer value for %s: %s", name, value)
        return default
    return parsed if parsed > 0 else default


DEFAULT_DOCUMENT_SCHEDULE_LOCK_TIMEOUT_SECONDS = _positive_int_env(DOCUMENT_SCHEDULE_LOCK_TIMEOUT_ENV, 600)
DEFAULT_DOCUMENT_SCHEDULE_LOCK_BLOCKING_TIMEOUT_SECONDS = _positive_int_env(
    DOCUMENT_SCHEDULE_LOCK_BLOCKING_TIMEOUT_ENV,
    30,
)


class DocumentScheduleLockError(RuntimeError):
    def __init__(self, document_id: str):
        super().__init__(f"Document '{document_id}' is busy with another scheduling operation")
        self.document_id = document_id


def contains_document_schedule_lock_error(errors) -> bool:
    return any(DOCUMENT_SCHEDULE_LOCK_BUSY_FRAGMENT in str(error) for error in errors or [])


def document_schedule_lock_failed_without_success(success_count: int, errors) -> bool:
    return success_count == 0 and contains_document_schedule_lock_error(errors)


def document_schedule_lock_key(document_id: str) -> str:
    return f"{DOCUMENT_SCHEDULE_LOCK_PREFIX}:{document_id}"


def _default_lock_factory(lock_key: str, *, timeout: int, blocking_timeout: int):
    from rag.utils.redis_conn import RedisDistributedLock

    return RedisDistributedLock(lock_key, timeout=timeout, blocking_timeout=blocking_timeout)


def normalize_document_ids(document_ids) -> list[str]:
    if document_ids is None:
        return []

    normalized = set()
    for document_id in document_ids:
        if document_id is None:
            continue
        document_id = str(document_id).strip()
        if document_id:
            normalized.add(document_id)
    return sorted(normalized)


class DocumentScheduleLocks:
    """Acquire per-document scheduling locks in stable order."""

    def __init__(
        self,
        document_ids,
        *,
        lock_factory=None,
        timeout: int | None = None,
        blocking_timeout: int | None = None,
    ):
        self.document_ids = normalize_document_ids(document_ids)
        self.lock_factory = lock_factory or _default_lock_factory
        self.timeout = timeout or DEFAULT_DOCUMENT_SCHEDULE_LOCK_TIMEOUT_SECONDS
        self.blocking_timeout = blocking_timeout or DEFAULT_DOCUMENT_SCHEDULE_LOCK_BLOCKING_TIMEOUT_SECONDS
        self._locks = []

    def __enter__(self):
        try:
            for document_id in self.document_ids:
                lock = self.lock_factory(
                    document_schedule_lock_key(document_id),
                    timeout=self.timeout,
                    blocking_timeout=self.blocking_timeout,
                )
                if not lock.acquire():
                    raise DocumentScheduleLockError(document_id)
                self._locks.append(lock)
        except Exception:
            self._release_acquired()
            raise
        return self

    def __exit__(self, exc_type, exc, tb):
        self._release_acquired()
        return False

    def _release_acquired(self):
        while self._locks:
            lock = self._locks.pop()
            try:
                lock.release()
            except Exception:
                logger.warning("Failed to release document scheduling lock", exc_info=True)


def document_schedule_locks(
    document_ids,
    *,
    lock_factory=None,
    timeout: int | None = None,
    blocking_timeout: int | None = None,
) -> DocumentScheduleLocks:
    return DocumentScheduleLocks(
        document_ids,
        lock_factory=lock_factory,
        timeout=timeout,
        blocking_timeout=blocking_timeout,
    )
