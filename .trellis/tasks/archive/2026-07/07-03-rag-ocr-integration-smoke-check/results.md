# RAG/OCR integration smoke results

Date: 2026-07-03
Branch: `Fisherman6a/test/rag-ocr-integration-smoke`

## 2026-07-04 resync startup findings

After resyncing to `upstream/develop` at `aadd57ab`, rerunning
`./scripts/local/dev-up.sh --china` exposed three separate startup issues:

1. Knowledge runtime artifact download failed before Docker work:
   - failing URL:
     `http://mirrors.tuna.tsinghua.edu.cn/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb`
   - root cause: `ragflow_deps/download_deps.py` globally forced
     `User-Agent: Mozilla/5.0`; the TUNA mirror returned `403 Forbidden` for
     that UA, while default urllib/curl-style headers could fetch the file.
   - final local fix: kept the single `Mozilla/5.0` UA strategy and changed the
     `--china` Ubuntu/Ubuntu-ports libssl URLs from TUNA to Huawei Cloud
     mirrors, which accept `Mozilla/5.0` for both amd64 and arm64 artifacts.
     Cache-first logging was also kept so existing artifacts print
     `Using cached ...` instead of a misleading `Downloading ...`.
   - verification: a direct `download_file` probe downloaded
     `libssl-test.deb` with size `1318204`; a direct `Mozilla/5.0` probe
     confirmed Huawei Cloud amd64 and arm64 libssl URLs return `200` with deb
     headers. Runtime deps then downloaded libssl, Tika, tokenizer, Chrome, uv
     release, NLTK, and HuggingFace `InfiniFlow/text_concat_xgb_v1.0` /
     `InfiniFlow/deepdoc` resources.

2. Docker image pull failed after runtime deps succeeded:
   - failing image:
     `docker.m.daocloud.io/docker.elastic.co/elasticsearch/elasticsearch:8.15.3`
   - Docker error:
     manifest HEAD to DaoCloud returned `403 Forbidden`.
   - official image path worked on this machine:
     `docker.elastic.co/elasticsearch/elasticsearch:8.15.3`.
   - workaround command used:

     ```bash
     GOPROXY=https://goproxy.cn,direct \
     GOSUMDB=sum.golang.google.cn \
     ./scripts/local/dev-up.sh --skip-knowledge-runtime-deps
     ```

     This keeps Go module downloads on the China mirror but avoids the broken
     DaoCloud Elasticsearch rewrite. With this command, Postgres, Redis, MinIO,
     and Elasticsearch pulled/started healthy.

3. Local demo seed was not idempotent after prior AI Gateway local credential
   rotation:
   - failing file/line:
     `deploy/seeds/002-ai-gateway-model-profiles.sql:153`
   - error:
     `duplicate key value violates unique constraint "uniq_provider_credentials_active_profile"`
     for `profile_id=default-chat`.
   - root cause: static seed tried to reactivate `cred-local-*` even when a
     different active credential already existed for the same default profile.
   - local fix: static seed now preserves existing active credentials and keeps
     `cred-local-*` rotated when another active credential exists.

Validation run on 2026-07-04:

```bash
python3 -m unittest \
  scripts.tests.test_local_seed_contract \
  scripts.tests.test_ai_gateway_local_seed_renderer \
  scripts.tests.test_knowledge_runtime_dependency_split -v

git diff --check

GOPROXY=https://goproxy.cn,direct \
GOSUMDB=sum.golang.google.cn \
./scripts/local/dev-up.sh --skip-knowledge-runtime-deps
```

Result: all checks passed and `dev-up` completed through infra health,
migrations, demo seed, and AI Gateway local seed overlay.

## Local State Verified

- Docker infra is running and healthy:
  - `software-teamwork-local-postgres-1` from `docker.m.daocloud.io/library/postgres:16-alpine`
  - `software-teamwork-local-redis-1` from `docker.m.daocloud.io/library/redis:7-alpine`
  - `software-teamwork-local-qdrant-1` from `docker.m.daocloud.io/qdrant/qdrant:v1.18.2`
  - `software-teamwork-local-minio-1` from `docker.m.daocloud.io/minio/minio:RELEASE.2025-09-07T16-13-09Z`
  - extra runtime doc engine `software-teamwork-local-elasticsearch` from `docker.elastic.co/elasticsearch/elasticsearch:8.19.3`
- Host services are reachable:
  - Gateway `http://127.0.0.1:8080/readyz`
  - Knowledge adapter `http://127.0.0.1:8083/readyz`
  - QA `http://127.0.0.1:8084/readyz`
  - AI Gateway `http://127.0.0.1:8086/readyz`
  - Knowledge runtime `http://127.0.0.1:9380/api/v1/system/healthz`
- Local OpenAI-compatible stub is running at `http://127.0.0.1:11434/v1`.
- Ignored local `deploy/.env` changes used for smoke:
  - `QA_SETTINGS_OPEN=true`
  - `KNOWLEDGE_AUTO_START_INGESTION=true`
  - local embedding provider pointed at `http://127.0.0.1:11434/v1`
  - `PADDLEOCR_BASE_URL`, `PADDLEOCR_API_URL`, `PADDLEOCR_ACCESS_TOKEN`, and `PADDLEOCR_ALGORITHM` remain empty.

## Resource Sources

- Knowledge runtime Python env: `services/knowledge-runtime/.venv`, created with `uv sync --python 3.13 --frozen`.
- Knowledge runtime API was started with `.venv/bin/python api/ragflow_server.py`; `uv run` was avoided because it tried to re-fetch `en-core-web-sm` from GitHub.
- Knowledge runtime worker was started with `.venv/bin/python rag/svr/task_executor.py -i 0 -t common`.
- DeepDOC resources were downloaded from HuggingFace mirror `https://hf-mirror.com/InfiniFlow/deepdoc` into `services/knowledge-runtime/rag/res/deepdoc`.
- NLTK resources required by runtime tokenization were installed under `/home/bao/nltk_data`:
  - `punkt_tab`
  - `wordnet`

## Workarounds Applied

- AI Gateway default profiles were placeholder credentials, so `/readyz` returned `503`.
  - Rotated `default-chat`, `default-embedding`, and `default-rerank` through `PATCH /internal/v1/model-profiles/{profileId}` to the local OpenAI-compatible stub.
  - After rotation, AI Gateway `/readyz` returned `200`.
- Knowledge runtime gateway-tenant auto-provisioning currently logs:
  - `load_user from gateway tenant header failed: Attempting to close database while transaction is open.`
  - Manually provisioned runtime tenants for smoke users, including `usr_local_admin`, then ran `init_env_default_models_for_tenant`.

## Smoke Results

### Knowledge ingestion real deps

Command:

```bash
cd services/knowledge
set -a; . ../../deploy/.env; set +a
KNOWLEDGE_INGESTION_SMOKE=1 \
KNOWLEDGE_SERVICE_BASE_URL=http://127.0.0.1:8083 \
KNOWLEDGE_INGESTION_SMOKE_TIMEOUT=4m \
go test ./internal/integration -run '^TestKnowledgeIngestionRealDepsSmoke$' -count=1 -v
```

Result: pass.

Coverage proved:

- Knowledge adapter
- Knowledge runtime API
- runtime worker
- Postgres
- Redis
- MinIO
- Elasticsearch
- local embedding provider stub
- Markdown upload, parse, chunk, embed, index, retrieval

### Gateway -> Knowledge -> QA RAG

As-is command:

```bash
cd services/knowledge
set -a; . ../../deploy/.env; set +a
GATEWAY_RAG_E2E_SMOKE=1 \
GATEWAY_BASE_URL=http://127.0.0.1:8080 \
VENDOR_RUNTIME_URL=http://127.0.0.1:9380 \
KNOWLEDGE_SERVICE_BASE_URL=http://127.0.0.1:8083 \
QA_SERVICE_BASE_URL=http://127.0.0.1:8084 \
AI_GATEWAY_BASE_URL=http://127.0.0.1:8086 \
KNOWLEDGE_TEST_DATABASE_URL='postgres://knowledge_app:knowledge_app_dev@127.0.0.1:5432/knowledge_system?sslmode=disable' \
KNOWLEDGE_REDIS_ADDR=127.0.0.1:6379 \
GATEWAY_SMOKE_USERNAME=admin \
GATEWAY_SMOKE_PASSWORD='<local admin password>' \
QA_SMOKE_CHAT_PROFILE_ID=default-chat \
QA_SMOKE_CHAT_MODEL=local-placeholder-chat \
GATEWAY_RAG_SMOKE_TIMEOUT=4m \
go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v
```

Result as-is: fail, not because the runtime chain is down.

- First failure before runtime tenant provisioning: `HTTP 401`.
- After tenant provisioning: `HTTP 502` from Runtime validation because the test helper sends `docType: "SMOKE"`, which Runtime rejects. Runtime accepts parser/chunk methods such as `naive`, `manual`, `qa`, `paper`, etc.
- After a temporary local test-only change from `docType: "SMOKE"` to `docType: "naive"`, the test reached retrieval but failed on a stale trace assertion. Adapter currently maps runtime trace as `EmbeddingDimension: 0` and `QdrantCollection: "elasticsearch"`.
- After temporarily relaxing that stale assertion, the full Gateway RAG E2E passed in `3.66s`, including upload, ingestion, retrieval, QA answer, and citations.

These temporary changes were restored. The git worktree was returned to clean state.

### Image-only PDF OCR smoke

Manual smoke generated an image-only PDF containing marker `OCR-GATEWAY-917` using Pillow, then used Gateway to create a `docType=naive` KB, upload the PDF, wait for ingestion, query the marker, and clean up.

Result: pass.

Observed output:

```text
ocr doc final ready chunks=1 parser=naive
ocr query status=201 hitCount=1 results=1 marker_found=True trace_backend=elasticsearch
```

This proves the local default builtin/DeepDOC path can OCR a simple image-only PDF fixture and retrieve the extracted marker through Gateway -> Knowledge -> Runtime.

This does not prove PaddleOCR cloud integration.

### QA MCP RAG smoke

Command:

```bash
set -a; . deploy/.env; set +a
GATEWAY_BASE_URL=http://127.0.0.1:8080 \
QA_SERVICE_BASE_URL=http://127.0.0.1:8084 \
KNOWLEDGE_SERVICE_BASE_URL=http://127.0.0.1:8083 \
GATEWAY_SMOKE_USERNAME=admin \
GATEWAY_SMOKE_PASSWORD='<local admin password>' \
bash scripts/run_issue_125_smoke.sh --qa-rag
```

Result: skipped, not failed.

- `knowledge_tool_available` skipped because no seed knowledge base was present after cleanup.
- QA message tests skipped because `QA_MCP_RAG_REAL_PROVIDER=1` was not set. The local AI Gateway stub is protocol-shaped, not a real provider quality check.

## Current OCR/PaddleOCR Status

- Default parser config:
  - id `parser_config_builtin_default`
  - backend `builtin`
  - `paddleocrAccessTokenConfigured=false`
- Current local env has no PaddleOCR cloud configuration:
  - `PADDLEOCR_BASE_URL` empty
  - `PADDLEOCR_API_URL` empty
  - `PADDLEOCR_ACCESS_TOKEN` empty
  - `PADDLEOCR_ALGORITHM` empty
- Local image-only PDF OCR via default builtin/DeepDOC works for the generated fixture.
- PaddleOCR cloud path is blocked until these are supplied:
  - `PADDLEOCR_BASE_URL` or `PADDLEOCR_API_URL`
  - `PADDLEOCR_ACCESS_TOKEN`
  - optional `PADDLEOCR_ALGORITHM`, default `PaddleOCR-VL`
  - a `paddleocr_cloud` parser config or env-driven runtime model provisioning.

## Known Code/Test Gaps Found

- `services/knowledge/internal/integration/gateway_smoke_helpers_test.go` uses `docType: "SMOKE"`, but current Runtime rejects that as an invalid `chunk_method`.
- `services/knowledge/internal/integration/gateway_rag_e2e_smoke_test.go` asserts `EmbeddingDimension > 0`, while current adapter intentionally maps runtime vendor trace with `EmbeddingDimension: 0` and `QdrantCollection: "elasticsearch"`.
- Knowledge runtime gateway-tenant auto-provisioning has a local code/runtime bug around Peewee transaction handling:
  - `Attempting to close database while transaction is open.`

## Useful Logs

- `.local/logs/knowledge.log`
- `.local/logs/ai-gateway.log`
- `.local/logs/qa.log`
- `services/knowledge-runtime/logs/ragflow_server.log`
- `services/knowledge-runtime/logs/task_executor_common_0.log`
