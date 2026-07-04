# Remove legacy AI provider compatibility

## Goal

Remove the old direct-provider compatibility paths and glue code that remain
after AI Gateway became the single model/provider exit. The target state is
simple and strict: AI model/provider calls outside `services/ai-gateway` use
AI Gateway profiles and internal AI Gateway APIs only.

This is not another public `services/gateway` routing task. Public Gateway
business API behavior should remain unchanged except for any schema/docs wording
that still implies domain services can store provider endpoints or API keys.

## Problem Statement

The previous task made the normal path use AI Gateway, but intentionally left
compatibility and emergency fallback code:

- QA still has legacy aggregate settings fields and code paths for
  `provider=direct`, `apiEndpoint`, encrypted API keys, and `apiKeyLast4`.
- Knowledge runtime still exposes direct model provider factories and model
  runtime DB/config paths that can resolve non-AI-Gateway providers.
- Repository policy still allowlists Knowledge runtime direct provider code.
- Tests, docs, and local scripts still contain examples such as
  `@SILICONFLOW`, `KNOWLEDGE_RUNTIME_MODEL_API_KEY`, or direct provider fallback
  wording.

The user clarified that "cleanup" means removing those compatibility surfaces,
not keeping old direct-provider paths as fallback glue.

## Confirmed Facts

- `.trellis/spec/backend/api-contracts.md:879` currently says direct provider
  SDK imports, provider base URLs, and non-internal OpenAI-compatible endpoints
  outside `services/ai-gateway` are forbidden unless allowlisted.
- `.trellis/spec/backend/api-contracts.md:922` still allows Knowledge runtime
  direct provider factories as explicit local/emergency fallbacks; this task
  must remove that exception.
- `docs/services/ai-gateway/docs/model-provider-exit-inventory.md:26` marks QA
  legacy aggregate settings as a cleanup candidate.
- `docs/services/ai-gateway/docs/model-provider-exit-inventory.md:31` marks
  Knowledge runtime direct provider factories as temporary fallback; this task
  must convert that from "temporary" to removed/forbidden.
- `services/qa/internal/service/settings.go:20` defines
  `llmProviderLegacyDirect = "direct"`.
- `services/qa/internal/service/settings.go:422` and
  `services/qa/internal/service/settings.go:439` still read legacy direct LLM
  config rows.
- `services/qa/internal/service/settings.go:490` still accepts `apiEndpoint`
  on legacy settings updates, even though new writes store AI Gateway profile
  semantics.
- `services/qa/internal/service/settings_types.go:87`,
  `services/qa/internal/service/settings_types.go:93`, and
  `services/qa/internal/service/settings_types.go:94` still expose legacy
  response fields in the aggregate settings DTO shape.
- `services/qa/internal/service/settings_types.go:106` and
  `services/qa/internal/service/settings_types.go:107` still accept legacy
  update fields.
- `services/qa/internal/service/settings_types.go:204` and
  `services/qa/internal/service/settings_types.go:205` still define legacy LLM
  connection-test input as endpoint/API-key based.
- `services/qa/internal/repository/settings_map.go:89` maps legacy
  `api_endpoint`, `api_key_encrypted`, and `api_key_last4` columns.
- `services/qa/internal/repository/settings_postgres.go:120` writes
  `api_endpoint`, `api_key_encrypted`, and `api_key_last4` columns as NULL for
  new legacy settings rows.
- `config/base.yaml:348` and `config/base.yaml:354` already default Knowledge
  runtime embedding/rerank factories to `AI_GATEWAY`.
- `config/base.yaml:360` and `config/base.yaml:362` already set
  `KNOWLEDGE_VENDOR_*_ID` to `@AI_GATEWAY`.
- `config/base.yaml:364` still exposes `KNOWLEDGE_RUNTIME_MODEL_API_KEY`.
- `services/knowledge-runtime/README.md:57` documents `SILICONFLOW` direct
  provider factories as explicit local/emergency fallback.
- `services/knowledge-runtime/api/utils/runtime_model_config.py:96` only reads
  `KNOWLEDGE_RUNTIME_MODEL_API_KEY` when the selected provider is not
  `AI_GATEWAY`.
- `services/knowledge-runtime/api/db/services/runtime_llm_service.py:142` can
  instantiate runtime model providers by factory name from runtime DB config.
- `services/knowledge/internal/adapterconfig/config.go:58` and
  `services/knowledge/internal/adapterconfig/config.go:59` forward configured
  vendor model IDs to runtime.
- `services/knowledge/internal/adapter/map_test.go:74`,
  `services/knowledge/internal/adapter/map_test.go:330`, and
  `services/knowledge/internal/adapter/map_test.go:454` still use
  `@SILICONFLOW` in test examples.
- `scripts/check_ai_gateway_provider_policy.py:49` allowlists
  `services/knowledge-runtime/rag/llm/` as direct provider fallback code.

## Confirmed Decisions

- Development-stage data loss is acceptable for this cleanup. QA schema should
  be simplified directly by editing the current table/migration definitions.
- Do not add extra compatibility/drop migrations just to preserve old
  endpoint/API-key fields. Local dev databases or data volumes can be cleared
  and recreated.
- Existing QA rows that depend on direct-provider columns or
  `provider=direct` are not a supported compatibility target for this task.

## Requirements

- R1. Remove QA aggregate-settings direct-provider compatibility from active
  code paths. QA settings must accept and expose only AI Gateway profile
  semantics for model configuration.
- R2. Remove QA endpoint/API-key connection-test semantics. Connection tests
  must validate an AI Gateway profile reference, not caller-provided provider
  URLs or direct API keys.
- R3. Stop reading or decrypting legacy QA `provider=direct` rows in runtime
  configuration. Existing local legacy rows are not supported; development
  databases should be rebuilt instead of adapted.
- R4. Remove Knowledge runtime direct model-provider fallback from normal
  product/runtime paths. Embedding and rerank factories must resolve to
  `AI_GATEWAY` only for this product integration.
- R5. Remove `KNOWLEDGE_RUNTIME_MODEL_API_KEY` and `@SILICONFLOW` direct
  provider examples from active config, scripts, runbooks, and project docs.
- R6. Tighten the provider policy checker so new direct provider code outside
  `services/ai-gateway` fails without a Knowledge runtime fallback allowlist.
- R7. Update tests so they assert AI Gateway-only behavior and do not normalize
  old direct-provider examples as acceptable.
- R8. Keep AI Gateway's own provider adapter and local seed overlay intact.
  Provider base URLs and provider API keys still belong in AI Gateway profiles.
- R9. Keep public `services/gateway` business route behavior unchanged.
- R10. Do not leak provider secrets, service tokens, prompts, raw provider
  errors, object keys, internal URLs, embedding vectors, or rerank payloads in
  logs, errors, docs, tests, or final reports.
- R11. Remove QA legacy direct-provider columns from the current schema
  definitions instead of adding schema compatibility layers or cleanup-only
  migrations.

## Acceptance Criteria

- [ ] AC1. `rg "llmProviderLegacyDirect|provider.*direct|apiEndpoint|apiKeyLast4|APIKeyEncrypted|APIKeyLast4" services/qa/internal/service services/qa/internal/repository services/qa/migrations` has no active legacy QA settings or schema path.
- [ ] AC2. QA aggregate settings DTOs and public/internal OpenAPI use
  `provider=ai-gateway`, `profileId`, model label, timeout, temperature, and
  max token fields only for LLM settings.
- [ ] AC3. QA LLM connection tests take a profile-based request and call AI
  Gateway validation/invocation, not arbitrary endpoint/API-key input.
- [ ] AC4. Knowledge runtime product default and allowed runtime path for
  embedding/rerank is AI Gateway only; direct provider factories are not
  documented or allowlisted as local/emergency fallback.
- [ ] AC5. Policy check fails on direct provider SDK/base URL/OpenAI-compatible
  provider endpoints under Knowledge runtime unless the path is inside
  `services/ai-gateway` or a test that explicitly asserts rejection.
- [ ] AC6. Config, `.env.example`, runbooks, service docs, and inventory no
  longer tell users to set `KNOWLEDGE_RUNTIME_MODEL_API_KEY` or
  `@SILICONFLOW` for product model calls.
- [ ] AC7. Existing AI Gateway local seed overlay remains supported for writing
  provider credentials into AI Gateway profiles.
- [ ] AC8. Gateway public route tests still pass, and no Gateway production code
  calls a provider directly.
- [ ] AC9. Targeted tests pass for QA, AI Gateway, Knowledge, Knowledge runtime
  policy/config utilities, and repository provider policy checks.
- [ ] AC10. Final worktree has no untracked runtime artifacts from test runs.
- [ ] AC11. QA migrations create the simplified AI-Gateway-only settings schema
  from an empty database, with no new migration added solely to drop the old
  direct-provider columns.

## Out Of Scope

- Changing public business API routing in `services/gateway`.
- Removing AI Gateway provider adapters, AI Gateway provider profile CRUD, or
  AI Gateway local seed overlay.
- Removing non-model parser/OCR integrations such as PaddleOCR or
  OpenDataLoader unless they are incorrectly wired as AI model-provider exits.
- Running real external provider smoke by default.
- Rewriting vendored Knowledge runtime provider catalog files that are inert
  metadata unless they are reachable by the product runtime path or policy
  check.
