from common.logging_utils import REDACTED, redact_text, sanitize_for_logging


def test_sanitize_for_logging_masks_nested_sensitive_keys():
    value = {
        "provider": {
            "api_key": "sk-secret",
            "base_url": "https://api.example/v1",
            "nested": [{"refresh_token": "refresh-secret"}],
        },
        "token": "session-token",
        "enabled": True,
    }

    sanitized = sanitize_for_logging(value)

    assert sanitized["provider"]["api_key"] == REDACTED
    assert sanitized["provider"]["base_url"] == "https://api.example/v1"
    assert sanitized["provider"]["nested"][0]["refresh_token"] == REDACTED
    assert sanitized["token"] == REDACTED
    assert sanitized["enabled"] is True


def test_sanitize_for_logging_masks_common_sensitive_header_keys():
    value = {
        "headers": {
            "api-key": "legacy-secret",
            "x-api-key": "header-secret",
            "access-token": "access-secret",
            "refresh-token": "refresh-secret",
            "proxy-authorization": "proxy-secret",
            "set-cookie": "session=secret",
            "content-type": "application/json",
        },
    }

    sanitized = sanitize_for_logging(value)

    assert sanitized["headers"]["api-key"] == REDACTED
    assert sanitized["headers"]["x-api-key"] == REDACTED
    assert sanitized["headers"]["access-token"] == REDACTED
    assert sanitized["headers"]["refresh-token"] == REDACTED
    assert sanitized["headers"]["proxy-authorization"] == REDACTED
    assert sanitized["headers"]["set-cookie"] == REDACTED
    assert sanitized["headers"]["content-type"] == "application/json"


def test_sanitize_for_logging_redacts_url_credentials_and_sensitive_query_values():
    value = "mysql://user:pass@example.internal:3306/ragflow?api-key=abc&token=def&timeout=3#secret-fragment"

    assert (
        sanitize_for_logging(value)
        == f"mysql://example.internal:3306/ragflow?api-key={REDACTED}&token={REDACTED}&timeout=3"
    )


def test_redact_text_masks_embedded_url_credentials_and_sensitive_query_values():
    text = "failed to connect mysql://user:pass@example.internal:3306/ragflow?token=def&timeout=3; retrying"

    redacted = redact_text(text)

    assert redacted == f"failed to connect mysql://example.internal:3306/ragflow?token={REDACTED}&timeout=3; retrying"
    assert "user:pass" not in redacted
    assert "def" not in redacted


def test_redact_text_masks_inline_assignments_and_bearer_tokens():
    text = "password=hunter2 token:abc Authorization: Bearer eyJhbGciOi secret=raw OAuth code=oauth-secret"

    redacted = redact_text(text)

    assert "hunter2" not in redacted
    assert "abc" not in redacted
    assert "eyJhbGciOi" not in redacted
    assert "raw" not in redacted
    assert "oauth-secret" not in redacted
    assert redacted.count(REDACTED) >= 5


def test_sanitize_for_logging_describes_bytes_without_content():
    assert sanitize_for_logging(b"secret-bytes") == "<12 bytes>"
