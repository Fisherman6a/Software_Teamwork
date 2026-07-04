# Implement: Knowledge bulk CSV upload

## Preconditions

- Work in `/Users/fengling/github/Software_Teamwork/.worktrees/knowledge-bulk-csv-upload`.
- Active task remains in `planning` until these artifacts are reviewed and
  `task.py start` is run.
- Inline Codex mode skips JSONL context curation; Phase 2 must load
  `trellis-before-dev` before code edits.

## Checklist

- [x] Re-read pre-development specs through `trellis-before-dev` before code
  edits.
- [x] Update `docs/services/gateway/api/public.openapi.yaml`:
  add `POST /api/v1/knowledge-bases/{knowledgeBaseId}/document-batches` and
  batch response schemas.
- [x] Update `services/knowledge/api/openapi.yaml` and, if present/relevant,
  `docs/services/knowledge/api/internal.openapi.yaml` for the matching internal
  route and schemas.
- [x] Update `docs/services/gateway/docs/active-api-owner-map.md` or the
  generation/audit source if project tooling requires it.
- [x] Regenerate frontend Gateway types with
  `bun run --cwd apps/web api:generate`.
- [x] Gateway backend:
  add active proxy route, permission configuration, downstream path mapping, and
  route-specific batch upload body limit.
- [x] Gateway tests:
  route/OpenAPI operation ID coverage, body-limit tests, proxy path test, and
  permission enforcement if not covered by existing table tests.
- [x] Knowledge backend:
  add batch route, multipart parser, per-file validation, sequential upload loop,
  partial-success aggregation, CSV content-type inference coverage, and
  sanitized per-file errors.
- [x] Knowledge tests:
  all-success batch, CSV content type, partial success, no files, too many
  files, empty file, and parse-start failure cleanup for only the failed file.
- [x] Frontend API:
  add `uploadDocumentBatch` wrapper and typed response.
- [x] Frontend hook:
  add `useUploadDocumentBatch` and invalidate knowledge document/base queries
  when `successCount > 0`.
- [x] Frontend page:
  replace single selected file state with upload item list, `multiple` input,
  multi-drop support, client-side rejection feedback, one batch submit, per-file
  results, failed-item retry behavior, and all-success reset.
- [x] Frontend tests:
  update old "uses only the first file" test, add multi-file/CSV/mixed
  validation/batch request/partial failure/invalidation cases.
- [x] Review generated OpenAPI type diffs and keep generated files unedited by
  hand.
- [x] Run validation commands.

## Validation Commands

Backend:

```bash
cd services/gateway && go test ./...
cd services/knowledge && go test ./...
```

Frontend:

```bash
bun run --cwd apps/web check
bun run --cwd apps/web build
bun run --cwd apps/web test:unit
```

Repository:

```bash
git diff --check
```

If validation scope is narrowed during implementation, report the skipped
commands and reason.

## Validation Record

- `cd services/gateway && go test ./...`
- `cd services/gateway && go build ./cmd/server`
- `cd services/knowledge && go test ./...`
- `cd services/knowledge && go build ./cmd/adapter`
- `bun run --cwd apps/web check`
- `bun run --cwd apps/web build`
- `bun run --cwd apps/web test:unit`
- `python3 scripts/verify_gateway_active_api.py`
- `python3 -m unittest scripts.tests.test_verify_gateway_active_api`
- `git diff --check`

## Risk / Rollback Points

- Contract changes come first. If generated frontend types expose an awkward
  shape, adjust OpenAPI before changing UI logic.
- Route-specific Gateway body limit is a distinct rollback point from Knowledge
  handler behavior.
- Keep the existing single upload route untouched so reverting the batch route
  does not break current users.
- Avoid changing global upload size defaults or Knowledge runtime behavior.
