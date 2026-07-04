# RAG OCR integration smoke check design

## Scope

This task is a diagnostic smoke run, not a feature implementation. It should
use repository scripts and tests to determine whether the local machine can run
the RAG/OCR path.

## Chain Under Test

The complete target chain is:

```text
Docker infra
  -> host-run Auth/File/Knowledge/AI Gateway/QA/Document/Gateway
  -> host-run Knowledge RAGFlow runtime API and worker
  -> runtime doc engine and embedding provider
  -> optional OCR provider such as PaddleOCR
  -> Gateway public Knowledge/QA APIs
```

The existing Markdown RAG smoke covers ingestion, chunking, embedding, indexing,
retrieval, QA answer, and citation when dependencies exist. It does not verify
OCR because the fixture is Markdown.

## Verification Strategy

1. Verify branch, clean worktree, and local config keys without exposing secret
   values.
2. Verify Docker infra health.
3. Start or verify host-run backend services using documented scripts.
4. Check Gateway, Knowledge adapter, QA, AI Gateway, and Knowledge runtime
   readiness.
5. Run existing safe smoke slices:
   - `bash scripts/run_issue_125_smoke.sh --qa-rag` for seeded KB/tool
     availability and optional QA RAG provider path.
   - `KNOWLEDGE_INGESTION_SMOKE=1 ... TestKnowledgeIngestionRealDepsSmoke`
     only if runtime ingestion prerequisites are present.
   - `GATEWAY_RAG_E2E_SMOKE=1 ... TestGatewayRAGE2ESmoke` only if runtime,
     embedding, AI Gateway chat provider, and QA settings prerequisites are
     present.
6. For OCR specifically, inspect PaddleOCR/runtime config and only run an OCR
   parser check if endpoint/token and runtime dependencies are configured.

## Blocked vs Failed

Missing external dependencies are reported as blocked:

- Knowledge runtime API/worker not running.
- Elasticsearch/doc engine not configured.
- Embedding provider not configured.
- PaddleOCR endpoint/token not configured.
- AI Gateway real chat provider not configured.

Failures after all prerequisites are present are reported as code/runtime
failures with request id, service log path, and safe error summary.

## No Secret Handling

The report must not include bearer tokens, service tokens, API keys, MinIO
object keys, bucket names, raw provider bodies, raw MCP payloads, or full
uploaded document contents.
