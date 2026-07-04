# RAG OCR integration smoke check implementation plan

## Checklist

- [ ] Confirm branch and clean worktree.
- [ ] Record safe local configuration summary.
- [ ] Check Docker infra status.
- [ ] Run `./scripts/local/dev-up.sh` if infra/migrations/seed need refresh.
- [ ] Run `./scripts/local/run-backend.sh` and verify host-run services.
- [ ] Check `http://127.0.0.1:8080/healthz` and relevant `/readyz` endpoints.
- [ ] Check Knowledge runtime API at `http://127.0.0.1:9380/api/v1/system/healthz`.
- [ ] If runtime is absent, inspect `services/knowledge-runtime` prerequisites
      and report blocked dependencies instead of claiming OCR failure.
- [ ] Run `bash scripts/run_issue_125_smoke.sh --qa-rag` for the available QA
      RAG slice.
- [ ] Run Knowledge ingestion real-deps smoke only when
      `KNOWLEDGE_AUTO_START_INGESTION=true`, runtime API/worker, doc engine,
      Redis, MinIO, PostgreSQL, and embedding provider are available.
- [ ] Run Gateway RAG E2E smoke only when Knowledge ingestion prerequisites,
      QA settings write path, and AI Gateway chat provider are available.
- [ ] Check PaddleOCR/OCR readiness separately; run only if endpoint/token and
      runtime OCR parser configuration are present.
- [ ] Summarize pass/fail/blocked with exact next steps.

## Commands

```bash
git status --short --branch
docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh
curl --noproxy '*' -fsS http://127.0.0.1:8080/healthz
curl --noproxy '*' -fsS http://127.0.0.1:8080/readyz
curl --noproxy '*' -fsS http://127.0.0.1:9380/api/v1/system/healthz
bash scripts/run_issue_125_smoke.sh --qa-rag
```

Conditional commands:

```bash
cd services/knowledge
KNOWLEDGE_INGESTION_SMOKE=1 go test ./internal/integration -run '^TestKnowledgeIngestionRealDepsSmoke$' -count=1 -v

cd services/knowledge
GATEWAY_RAG_E2E_SMOKE=1 go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v
```

## Rollback

This task should not modify application code. If local services are started,
they can be stopped with:

```bash
./scripts/local/stop-backend.sh
```
