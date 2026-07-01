# S-042 AI Gateway Real Provider API Key Smoke

## Goal

Complete GitHub issue #378 by making the local AI Gateway real-provider path verifiable without committing secrets. The team should be able to answer: an API key alone is not enough; operators also need AI Gateway profiles, downstream service env/profile wiring, and explicit smoke validation.

## Confirmed Facts

- Work branch was refreshed onto latest `upstream/develop` at `83b01c1` after the branch advanced from the earlier `6913e52` and `0e05b32` baselines.
- Issue #378 is assigned to `AndyXuPrime`, status `In Progress`, target branch `develop`, suggested work branch `Special/test/real-provider-api-key-smoke`.
- `services/ai-gateway` already has chat, streaming chat, embedding, rerank adapters, credential encryption, invocation summaries, and an env-gated `AI_GATEWAY_REAL_PROVIDER_SMOKE=1` test entry.
- Root Compose `--profile ai` seeds enabled local placeholder profiles and fake encrypted credentials. These are useful for readiness/config checks but are not proof of real provider availability.
- Frontend must not call AI Gateway directly. Browser-visible APIs stay behind Gateway public `/api/v1/**`.

## Requirements

- Preserve secret boundaries: do not commit provider API keys, service tokens, private env files, full prompts, document bodies, embedding payloads, or provider raw error bodies.
- Update AI Gateway readiness so it distinguishes missing profiles, configured real/unknown credentials, and known local placeholder credentials for chat, embedding, and rerank purposes.
- Keep real provider tests env-gated. Ordinary `go test ./...` must skip external model calls unless the explicit smoke env is set.
- Document the real-provider setup path across AI Gateway, QA, Knowledge, and Document using existing docs/runbooks as the source of truth.
- Clarify that Document report generation validation covers report/job/event/section workflow and that rich DOCX Pandoc/LibreOffice runtime is out of scope.
- Clarify that Knowledge has two distinct paths: default local hashing/no-op rerank and optional AI Gateway embedding/rerank profiles.

## Acceptance Criteria

- [x] AI Gateway `/readyz` after `docker compose --profile ai up -d --build` reports distinct check statuses for missing profile, known local placeholder credential, and configured non-placeholder credential.
- [x] `AI_GATEWAY_REAL_PROVIDER_SMOKE=1` remains the explicit real-provider gate and can verify at least chat when real provider env is supplied; embedding/rerank run only when their model env vars are supplied.
- [x] Readiness and tests do not decrypt, log, or return provider API keys, service tokens, prompts, document text, embedding vectors, or provider raw bodies.
- [x] Docs explain API key alone is insufficient; operators must configure profiles, downstream env/profile IDs, service token/caller headers, and run smoke.
- [x] Cross-service acceptance docs cover Gateway/AI Gateway, QA message, Document `summer_peak_inspection` outline/content job workflow, and Knowledge local hashing vs real AI Gateway embedding/rerank.
- [x] Verification includes service tests/builds for changed Go code, docs whitespace checks, and Compose config checks when deploy docs/Compose expectations are touched.

## Out Of Scope

- Submitting any real API key, provider account details, `.env` private values, or production service tokens.
- Making real provider smoke a required CI check.
- Implementing new provider-specific non-OpenAI-compatible adapters.
- Changing frontend code to call AI Gateway directly.
- Implementing rich DOCX Pandoc/LibreOffice runtime.

## Open Questions

None blocking. The issue body and repository docs define the required scope.
