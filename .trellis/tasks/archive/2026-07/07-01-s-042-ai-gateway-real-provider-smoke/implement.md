# Implementation Plan

1. Inspect AI Gateway seed data and credential metadata to identify local placeholder fingerprints without decrypting credentials.
2. Update `services/ai-gateway/internal/service` readiness classification and add unit/handler tests for missing, placeholder, and non-placeholder credentials.
3. Update focused docs/runbooks for real provider setup and cross-service acceptance without broad rewrites.
4. Run formatting and service checks:
   - `cd services/ai-gateway && go test ./...`
   - `cd services/ai-gateway && go build ./cmd/server`
   - `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet`
   - `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example --profile ai config --quiet`
   - `git diff --check`
5. Run the env-gated real-provider smoke only if real provider env exists locally; otherwise verify the gate skips and report it as not run/skipped.

## Risk Points

- `/readyz` must not imply external provider acceptance. It can only classify profile/credential configuration.
- Documentation must not claim a complete one-click backend E2E exists before #125 lands.
- Avoid printing or adding real secret values to tests, docs, task artifacts, or shell output.
