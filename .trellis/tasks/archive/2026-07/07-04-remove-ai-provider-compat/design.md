# Design: Remove Legacy AI Provider Compatibility

## Boundary

The strict boundary is:

```text
domain services -> services/ai-gateway -> provider adapters
```

The following are no longer acceptable normal or fallback paths:

```text
qa/document/knowledge/gateway -> provider
knowledge-runtime product path -> provider
qa aggregate settings -> direct endpoint/api key
```

AI Gateway remains the only service that may store provider credentials, manage
provider base URLs, call OpenAI/SiliconFlow/local-compatible providers, normalize
provider errors, and record invocation/usage summaries.

## QA Cleanup Design

### Target Shape

QA aggregate settings should use the same model concept as the current LLM
config version API:

```json
{
  "llm": {
    "provider": "ai-gateway",
    "profileId": "default-chat",
    "model": "deepseek-ai/DeepSeek-V4-Flash",
    "timeoutSeconds": 60,
    "temperature": 0.7,
    "maxTokens": 1024
  }
}
```

Remove these active DTO fields from aggregate settings and update tests/docs:

- `apiEndpoint`
- `apiKey`
- `apiKeyConfigured`
- `apiKeyLast4`
- direct-provider `tokenHeader` if it is only meaningful for user-supplied
  provider credentials

### Runtime Behavior

`runtimeLLM` should no longer accept `provider=direct`. It should:

- use active `provider=ai-gateway` rows with `profile_id`,
- fall back to bootstrap AI Gateway config only when no row exists,
- fail with a clear configuration error when the active row is not
  `provider=ai-gateway`.

### Repository Shape

This is a development-stage cleanup with an accepted data reset. The QA schema
should be simplified directly in the current table/migration definitions:

- remove legacy `api_endpoint`, `api_key_encrypted`, `api_key_last4`, and
  direct-provider-only token header storage from active schema definitions,
- do not add a compatibility migration whose only purpose is preserving or
  dropping the old direct-provider fields,
- treat local dev database or volume rebuild as the migration path for stale
  rows.

The target is an empty-database schema that only represents AI-Gateway-backed
model settings.

## Knowledge Runtime Cleanup Design

### Product Runtime

The product integration should resolve embedding/rerank through:

```text
KNOWLEDGE_RUNTIME_EMBEDDING_FACTORY=AI_GATEWAY
KNOWLEDGE_RUNTIME_RERANK_FACTORY=AI_GATEWAY
KNOWLEDGE_VENDOR_EMBEDDING_ID=<model>@default@AI_GATEWAY
KNOWLEDGE_VENDOR_RERANK_ID=<model>@default@AI_GATEWAY
```

`KNOWLEDGE_RUNTIME_MODEL_API_KEY` should not be part of product docs or default
startup. AI Gateway service token/profile configuration replaces it.

### Runtime Provider Guard

For project-owned runtime config utilities, reject non-`AI_GATEWAY` factories
for embedding/rerank product paths. The most direct guard points are:

- `services/knowledge-runtime/api/utils/runtime_model_config.py`
- `services/knowledge-runtime/deploy/check_runtime_dependencies.py`
- startup scripts that validate runtime env

Vendored provider classes may remain in files if deleting them would be a
large vendor fork, but the product path and policy checker must no longer
allow them as runtime exits. If a file remains only as inert vendor catalog,
document it as not reachable by product config, not as a fallback.

## Policy Checker

Update `scripts/check_ai_gateway_provider_policy.py`:

- remove the broad `services/knowledge-runtime/rag/llm/` allowlist,
- keep `services/ai-gateway/` allowlisted,
- keep local AI Gateway seed scripts allowlisted,
- allow tests only when the test asserts rejection or AI Gateway adapter
  behavior,
- fail direct provider SDKs/base URLs in Knowledge runtime product paths.

## Documentation

Replace "temporary local/emergency fallback" wording with "removed/forbidden".
Keep only the AI Gateway seed overlay as the supported place where local
provider base URLs and provider keys are entered.

## Rollback

Rollback is ordinary git revert plus recreating the local development database
from the previous schema if a developer needs to inspect old local data.
