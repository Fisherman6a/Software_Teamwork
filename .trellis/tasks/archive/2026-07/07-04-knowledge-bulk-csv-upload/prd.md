# Knowledge bulk CSV upload

## Goal

Knowledge managers can submit multiple documents to one knowledge base in one
upload action, including CSV files, and receive per-file success/failure feedback
while the Knowledge ingestion pipeline queues successfully accepted documents.

## Background And Decisions

- Work is performed in the isolated git worktree
  `/Users/fengling/github/Software_Teamwork/.worktrees/knowledge-bulk-csv-upload`
  on branch `Sakayori-Iroha-168/feat/knowledge-bulk-csv-upload`.
- PR target branch is `develop`.
- The current browser-facing upload endpoint is
  `POST /api/v1/knowledge-bases/{knowledgeBaseId}/documents` in
  `docs/services/gateway/api/public.openapi.yaml`; it accepts one
  `multipart/form-data` file.
- Gateway currently proxies the upload route to Knowledge without business
  aggregation in `services/gateway/internal/http/routes.go` and
  `services/gateway/internal/http/proxy.go`.
- Knowledge's current internal upload contract is also one-document shaped:
  `POST /internal/v1/knowledge-bases/{knowledgeBaseId}/documents` in
  `services/knowledge/api/openapi.yaml`.
- Knowledge owns knowledge document upload, status, chunks, content, and
  runtime adapter behavior. Gateway must not parse, index, or call
  Knowledge runtime directly.
- Knowledge runtime/vendor client currently uploads one document per runtime
  request; backend batch upload will aggregate per-file results in the
  Knowledge adapter while still calling runtime once per file internally.
- CSV is already present in the frontend allowed extension/MIME/type label
  lists in `apps/web/src/pages/knowledge/documents/page.tsx`.
- CSV is already recognized by the Knowledge adapter as `text/csv` in
  `services/knowledge/internal/adapter/content_type.go`.
- Existing frontend upload tests explicitly assert the current limitation:
  multi-file drops use only the first file in
  `apps/web/src/pages/knowledge/documents/page.test.tsx`.
- Product decision: batch upload must extend through the backend public and
  internal contracts rather than only orchestrating multiple single-file
  requests in the frontend.
- Product decision: backend batch upload uses partial success semantics. A
  failed file must not roll back successfully uploaded files in the same batch.
- Product decision: the first implementation supports at most 10 files per
  batch. Existing per-file 32 MiB semantics remain; Gateway and Knowledge must
  allow the batch route to carry up to 10 files plus multipart overhead.

## Requirements

- R1. The knowledge documents upload dialog must accept selecting or dropping
  multiple files at once.
- R2. CSV files must be accepted through the same upload UI and remain displayed
  as CSV in document type labels.
- R3. Unsupported files in a selected batch must be rejected with actionable
  per-file feedback without discarding already selected valid files.
- R4. Upload submission must preserve the current shared tag behavior: the
  comma-separated tags entered in the dialog apply to every uploaded file.
- R5. The UI must show enough result state for users to understand which files
  were accepted and which failed.
- R6. Gateway OpenAPI must expose a backend batch-upload contract so the browser
  can submit a multi-file upload as one request to Gateway.
- R7. Gateway must route the batch-upload request to Knowledge and apply a
  route-specific body limit suitable for 10 files.
- R8. Knowledge internal OpenAPI and adapter routes must support the matching
  batch-upload contract.
- R9. The backend batch-upload response must include per-file results with
  filename, status, optional document summary, and optional sanitized error.
- R10. Partial successes must be durable: successful documents remain uploaded
  and, when auto-ingestion is enabled, enter parsing even if another file in the
  batch fails.
- R11. After any successful upload in a batch, document lists and knowledge-base
  counts must be invalidated/refetched as they are for the current single upload
  flow.
- R12. Existing permission checks must continue to gate the upload entry and
  backend route.
- R13. Existing single-file upload behavior must remain valid as the one-file
  case of the batch flow.

## Acceptance Criteria

- [ ] AC1. A user with upload permission can select/drop more than one supported
  file and sees all valid files listed before submission.
- [ ] AC2. A selected CSV file is accepted by client-side validation and sent in
  a backend batch upload request.
- [ ] AC3. A mixed client-side selection of supported and unsupported files keeps
  supported files selected and reports unsupported filenames/types.
- [ ] AC4. Clicking upload sends one Gateway batch request with all selected
  valid files, the selected `knowledgeBaseId`, and parsed tags.
- [ ] AC5. The backend returns per-file results using partial success semantics;
  successful files include document summaries and failed files include sanitized
  error codes/messages.
- [ ] AC6. Partial failures are visible per file; successful files stay marked
  successful and failed files can be retried without reselecting the successful
  batch.
- [ ] AC7. When all files in a batch succeed, the dialog closes or resets with a
  clear success notice, the page resets to page 1, and the document list refreshes.
- [ ] AC8. Existing document type display continues to render CSV documents as
  `CSV`.
- [ ] AC9. Backend contract and handler tests cover multi-file multipart upload,
  CSV content-type inference, 10-file limit validation, partial success response
  semantics, and validation for empty/no-file requests.
- [ ] AC10. Gateway tests cover the new active route, downstream path mapping,
  permission enforcement, and route-specific batch upload body limit.
- [ ] AC11. Frontend API and component tests cover multi-file selection/drop,
  CSV acceptance, unsupported-file feedback, one batch request, result display,
  invalidation after any success, and at least one partial failure path.

## Out Of Scope

- Changing Knowledge runtime parsing/chunking behavior for CSV content.
- Adding new drag-and-drop dependencies.
- Introducing a true runtime/vendor batch upload call.
- Changing global upload size configuration outside the new route-specific
  Gateway/Knowledge limits needed for the batch route.
- Full browser E2E against real Knowledge runtime dependencies; env-gated smoke
  tests remain available but are not required for this task's local completion.

## Open Questions

None. Planning decisions above are ready for technical design review.
