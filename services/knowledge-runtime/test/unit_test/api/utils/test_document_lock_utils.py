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
import pytest

from api.utils.document_lock_utils import (
    DocumentScheduleLockError,
    document_schedule_lock_failed_without_success,
    document_schedule_locks,
)


@pytest.mark.p2
def test_document_schedule_locks_dedupes_sorts_and_releases_in_reverse_order():
    events = []

    class FakeLock:
        def __init__(self, key):
            self.key = key

        def acquire(self):
            events.append(("acquire", self.key))
            return True

        def release(self):
            events.append(("release", self.key))

    def lock_factory(key, timeout, blocking_timeout):
        events.append(("create", key, timeout, blocking_timeout))
        return FakeLock(key)

    with document_schedule_locks(
        ["doc-b", "doc-a", "doc-b", "", None],
        lock_factory=lock_factory,
        timeout=7,
        blocking_timeout=3,
    ) as locks:
        assert locks.document_ids == ["doc-a", "doc-b"]
        events.append(("body",))

    assert events == [
        ("create", "document_schedule:doc-a", 7, 3),
        ("acquire", "document_schedule:doc-a"),
        ("create", "document_schedule:doc-b", 7, 3),
        ("acquire", "document_schedule:doc-b"),
        ("body",),
        ("release", "document_schedule:doc-b"),
        ("release", "document_schedule:doc-a"),
    ]


@pytest.mark.p2
def test_document_schedule_locks_releases_acquired_locks_when_later_acquire_fails():
    events = []

    class FakeLock:
        def __init__(self, key):
            self.key = key

        def acquire(self):
            events.append(("acquire", self.key))
            return self.key != "document_schedule:doc-b"

        def release(self):
            events.append(("release", self.key))

    def lock_factory(key, timeout, blocking_timeout):
        events.append(("create", key, timeout, blocking_timeout))
        return FakeLock(key)

    with pytest.raises(DocumentScheduleLockError) as exc_info:
        with document_schedule_locks(
            ["doc-b", "doc-a"],
            lock_factory=lock_factory,
            timeout=7,
            blocking_timeout=3,
        ):
            raise AssertionError("body should not run")

    assert exc_info.value.document_id == "doc-b"
    assert events == [
        ("create", "document_schedule:doc-a", 7, 3),
        ("acquire", "document_schedule:doc-a"),
        ("create", "document_schedule:doc-b", 7, 3),
        ("acquire", "document_schedule:doc-b"),
        ("release", "document_schedule:doc-a"),
    ]


@pytest.mark.p2
def test_document_schedule_locks_release_when_body_raises():
    events = []

    class FakeLock:
        def __init__(self, key):
            self.key = key

        def acquire(self):
            events.append(("acquire", self.key))
            return True

        def release(self):
            events.append(("release", self.key))

    def lock_factory(key, timeout, blocking_timeout):
        return FakeLock(key)

    with pytest.raises(ValueError):
        with document_schedule_locks(["doc-a"], lock_factory=lock_factory):
            raise ValueError("boom")

    assert events == [
        ("acquire", "document_schedule:doc-a"),
        ("release", "document_schedule:doc-a"),
    ]


@pytest.mark.p2
def test_document_schedule_lock_failed_without_success_only_matches_zero_success_lock_errors():
    busy_error = str(DocumentScheduleLockError("doc-a"))

    assert document_schedule_lock_failed_without_success(0, [busy_error]) is True
    assert document_schedule_lock_failed_without_success(1, [busy_error]) is False
    assert document_schedule_lock_failed_without_success(0, ["Document not found"]) is False
    assert document_schedule_lock_failed_without_success(0, []) is False
