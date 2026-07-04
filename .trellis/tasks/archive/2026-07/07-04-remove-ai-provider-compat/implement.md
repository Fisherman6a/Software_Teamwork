# Implement Plan: Remove Legacy AI Provider Compatibility

## Step 1: Apply Development-Stage Schema Cleanup

- Edit the current QA table/migration definitions directly so new development
  databases no longer create legacy direct-provider columns.
- Remove `api_endpoint`, `api_key_encrypted`, `api_key_last4`, and
  direct-provider-only token-header storage from the active schema shape.
- Do not add a compatibility/drop migration solely for the old fields.
- Expect local dev databases or data volumes to be cleared and recreated.

## Step 2: QA Direct Compatibility Removal

- Remove `llmProviderLegacyDirect`.
- Remove `APIEndpoint`, `APIKeyEncrypted`, `APIKeyLast4`,
  `APIKeyConfigured`, and update-only `APIKey` fields from active aggregate
  settings DTOs.
- Replace aggregate LLM connection-test input with profile-based input.
- Update `ConfigService.activeLLM`, `runtimeLLM`, `buildStoredLLM`, and
  audit-data helpers to reject non-`ai-gateway` configs instead of adapting
  them.
- Update repository mapping/inserts and generated query bindings to avoid
  direct-provider fields.
- Update QA settings tests and API docs.

## Step 3: Knowledge Runtime AI-Gateway-Only Enforcement

- Remove product docs and scripts that recommend
  `KNOWLEDGE_RUNTIME_MODEL_API_KEY` or `@SILICONFLOW`.
- Update Knowledge adapter tests to use `@AI_GATEWAY` examples.
- Add/update runtime config tests so non-`AI_GATEWAY` embedding/rerank factory
  values fail in project product mode.
- Keep AI Gateway runtime provider tests for `AIGatewayEmbed` and
  `AIGatewayRerank`.

## Step 4: Policy And Inventory

- Remove Knowledge runtime direct provider allowlist from
  `scripts/check_ai_gateway_provider_policy.py`.
- Update policy tests to assert Knowledge runtime direct provider code is
  reported unless the test explicitly checks AI Gateway adapter behavior or
  rejection.
- Update `docs/services/ai-gateway/docs/model-provider-exit-inventory.md`.
- Update `.trellis/spec/backend/api-contracts.md` to remove fallback language.

## Step 5: Validation

Run at minimum:

```bash
python3 scripts/check_ai_gateway_provider_policy.py
python3 -m unittest scripts.tests.test_ai_gateway_provider_policy

cd services/qa
go test ./...
go build ./cmd/server
go build ./cmd/agent

cd ../ai-gateway
go test ./...
go build ./cmd/server

cd ../gateway
go test ./...
go build ./cmd/server

cd ../knowledge
go test ./...
go build ./cmd/adapter

cd ../knowledge-runtime
PYTHONPATH=. uv run --no-project --with pytest --with pytest-asyncio python -m pytest \
  test/routes/test_runtime_dependency_check.py \
  test/unit_test/rag/llm/test_ai_gateway_provider.py \
  test/unit_test/api/utils/test_runtime_model_config.py \
  -q

git diff --check
```

If QA migration files change, run the relevant empty-database migration apply
check against a local PostgreSQL database or record why it could not run.

## Step 6: Review Checklist

- No active direct provider fallback outside `services/ai-gateway`.
- No public Gateway route behavior changes except schema text alignment.
- No docs tell users to configure direct provider keys in QA/Knowledge runtime.
- No tests normalize `@SILICONFLOW` as the product example.
- Worktree has no generated runtime artifacts.
