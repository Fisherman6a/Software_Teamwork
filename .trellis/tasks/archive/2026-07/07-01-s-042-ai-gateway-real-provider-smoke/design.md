# Design

## Boundaries

- AI Gateway owns model profiles, provider configuration, encrypted credentials, OpenAI-compatible chat/embedding/rerank endpoints, readiness, and sanitized invocation summaries.
- QA, Knowledge, and Document only reference AI Gateway profiles and model names. They do not store provider base URLs or provider API keys.
- Gateway remains the browser/public entrypoint. AI Gateway `/internal/v1/**` routes stay service-to-service only.

## Readiness

`Service.CheckReady` will keep its existing envelope and HTTP behavior. For each purpose it will classify the best enabled profile into one of:

- `ok`: enabled profile has a configured credential and it is not a known local placeholder.
- `placeholder`: enabled profile has a configured credential but its credential metadata matches the repository's seeded local placeholder credentials.
- `missing`: no enabled profile for the purpose, or no active credential is configured.

The service cannot prove a credential is accepted by the external provider without making a provider call. Therefore `/readyz` will identify known placeholders and configured credentials, while real external availability remains proven by env-gated smoke tests.

## Placeholder Detection

The seed script stores deterministic fake credential metadata. Readiness can compare low-sensitive `ProviderCredential.KeyLast4` and `FingerprintSHA256` metadata against known local placeholder values. The field name remains `FingerprintSHA256` for schema compatibility, but latest `develop` writes HMAC-SHA-256 fingerprints derived from the credential encryption key, not bare SHA-256. Readiness must not decrypt credentials.

If a profile has `APIKeyConfigured=true` but the active credential cannot be read, readiness should classify that purpose as missing/degraded rather than panic or leak repository errors.

## Docs And Smoke

Docs will preserve the existing runbook hierarchy:

- `deploy/README.md` for root Compose and local operator entrypoints.
- `docs/runbooks/local-integration.md` for cross-service validation workflow.
- `docs/testing/strategy.md` for env-gated smoke and PR verification expectations.
- `docs/services/ai-gateway/docs/seed-runbook.md` for profile/key creation and direct AI Gateway smoke.

## Rollback

The code change is limited to readiness classification and tests. Rolling back restores the previous boolean profile readiness behavior. Documentation updates can be reverted independently.
