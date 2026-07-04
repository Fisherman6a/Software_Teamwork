# RAG OCR integration smoke check

## Goal

Create a clean test branch and determine how far the local RAG/OCR chain can
run on this machine, using the repository's documented startup flow and
existing smoke tests before attempting ad hoc manual calls.

The useful output is an evidence-backed status: pass, fail, or blocked for each
layer in the chain, with concrete commands and intervention steps when a
dependency is missing.

## Background

- Branch: `Fisherman6a/test/rag-ocr-integration-smoke`, created from current
  `develop` at `e6ef4f15`.
- Local Docker infra is already running and healthy for PostgreSQL, Redis,
  Qdrant, and MinIO.
- Host-run backend services are not currently running; `.local/run/` has no pid
  files and Gateway `127.0.0.1:8080` is not listening.
- Knowledge runtime API is not currently running; `127.0.0.1:9380` is not
  listening.
- `deploy/.env` currently has:
  - `VENDOR_RUNTIME_URL=http://127.0.0.1:9380`
  - `KNOWLEDGE_AUTO_START_INGESTION=false`
  - `AI_GATEWAY_BASE_URL=http://localhost:8086`
  - `QA_SETTINGS_OPEN=false`
- `deploy/.env` does not currently define PaddleOCR endpoint/token keys or
  Knowledge runtime embedding provider keys.
- Repository docs say the default local stack starts only the Knowledge adapter;
  real parsing, chunking, embedding, indexing, and retrieval require explicitly
  starting `services/knowledge-runtime` API and worker plus external doc engine
  and embedding provider.
- Existing `GATEWAY_RAG_E2E_SMOKE` uses a Markdown fixture and explicitly does
  not require a real OCR model. It verifies Gateway -> Knowledge runtime
  ingestion/retrieval -> QA answer/citation when all runtime and model
  dependencies are available, but it does not prove OCR quality or PaddleOCR
  integration.
- Real OCR requires additional runtime/provider configuration, such as a
  reachable PaddleOCR-compatible endpoint/token and an OCR-capable parser path.

## Requirements

- Keep the current work on a dedicated test branch from latest `develop`.
- Follow the documented local path: Docker infra through `dev-up.sh`, host-run
  backend through `run-backend.sh`, and Knowledge runtime from
  `services/knowledge-runtime` when needed.
- Do not leak secrets, service tokens, bearer tokens, object keys, bucket names,
  raw provider bodies, prompts, or document text beyond minimal safe markers.
- First run low-risk readiness and existing smoke tests to narrow failures:
  backend readiness, Knowledge owner route, Knowledge ingestion real-deps smoke,
  Gateway RAG E2E smoke, and QA MCP RAG slice where prerequisites are present.
- Treat unavailable runtime, Elasticsearch/doc engine, embedding provider,
  PaddleOCR endpoint/token, or real AI Gateway chat provider as `blocked`, not
  as a business-code failure.
- If full OCR cannot run from current local config, provide exact missing
  environment variables and commands the user can run or provide.
- Avoid code changes unless a project script/test has an obvious local-only
  issue preventing the documented path from running.

## Acceptance Criteria

- [ ] Test branch exists and task metadata records it.
- [ ] Current local config and running-process state are recorded without
      leaking secrets.
- [ ] Docker infra status is checked.
- [ ] Backend startup/readiness is attempted or an existing running backend is
      verified.
- [ ] Knowledge runtime API/worker prerequisites are checked.
- [ ] Existing RAG/Knowledge smoke tests are run where prerequisites allow.
- [ ] OCR/PaddleOCR path is checked separately from the Markdown RAG smoke.
- [ ] Final result states exactly which layers passed, failed, or are blocked,
      with log paths and next commands for user intervention.

## Notes

- Primary docs inspected:
  - `README.md`
  - `deploy/README.md`
  - `docs/runbooks/local-integration.md`
  - `docs/runbooks/issue-125-smoke.md`
  - `docs/testing/strategy.md`
- Primary test entrypoints inspected:
  - `scripts/run_issue_125_smoke.sh`
  - `services/knowledge/internal/integration/ingestion_real_deps_smoke_test.go`
  - `services/knowledge/internal/integration/gateway_rag_e2e_smoke_test.go`
  - `services/deploy/smoke/mcp_qa_rag_smoke_test.go`
