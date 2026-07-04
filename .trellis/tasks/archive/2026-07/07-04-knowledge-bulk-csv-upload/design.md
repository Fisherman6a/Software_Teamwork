# Design: Knowledge bulk CSV upload

## Architecture

The feature spans Gateway public contract, Knowledge internal contract, the
Knowledge adapter, generated frontend types, and the knowledge documents page.

Data flow:

```text
apps/web KnowledgeDocumentsPage
  -> POST /api/v1/knowledge-bases/{knowledgeBaseId}/document-batches
  -> Gateway proxy route
  -> POST /internal/v1/knowledge-bases/{knowledgeBaseId}/document-batches
  -> Knowledge adapter loops over multipart files
  -> vendor UploadDocument once per file
  -> optional StartDocumentParse for each successful document
  -> aggregate per-file batch response
```

Gateway remains a public routing and authorization boundary. Knowledge owns the
domain behavior, runtime calls, document state, and per-file result aggregation.

## Contracts

### Public Gateway Route

Add:

```text
POST /api/v1/knowledge-bases/{knowledgeBaseId}/document-batches
```

Reason for path shape:

- It is a resource collection creation, not an action path.
- It does not change the existing single-document upload route.
- It keeps batch behavior explicit for generated frontend types and body-limit
  routing.

Request:

- `multipart/form-data`
- Repeated file field name: `files`
- Optional repeated shared tag field: `tags`
- Maximum 10 file parts.
- Empty files are per-file failures when other files are present; a request with
  no files is a request-level `400 validation_error`.

Response:

- `201 Created` when every file succeeds.
- `207 Multi-Status` when at least one file succeeds and at least one file
  fails.
- `400 validation_error` when the request shape is invalid before per-file
  processing can begin, including no files, too many files, or multipart parse
  failure.
- `502 dependency_error` only when the whole batch cannot be processed before
  any per-file result can be produced because a shared dependency is unavailable.
  Per-file runtime failures after parsing begins are represented in the batch
  result.

Envelope:

```json
{
  "data": {
    "totalCount": 2,
    "successCount": 1,
    "failedCount": 1,
    "results": [
      {
        "filename": "guide.pdf",
        "status": "uploaded",
        "document": { "...": "DocumentSummary" }
      },
      {
        "filename": "bad.pdf",
        "status": "failed",
        "error": {
          "code": "validation_error",
          "message": "vendor runtime validation failed"
        }
      }
    ]
  },
  "requestId": "req_123"
}
```

Per-file result status values:

- `uploaded`
- `failed`

The `error` object must not include object keys, internal URLs, stack traces,
provider raw responses, SQL, MinIO details, or runtime internals.

### Knowledge Internal Route

Add:

```text
POST /internal/v1/knowledge-bases/{knowledgeBaseId}/document-batches
```

The internal request/response shape mirrors Gateway's public contract. Gateway
proxies the response body and normalizes only route-level errors as it already
does for owner-service proxy routes.

## Backend Design

### Gateway

Changes:

- Add an active proxy route with owner `knowledge`, operation ID
  `uploadKnowledgeBaseDocumentBatch`, and the same knowledge write/admin
  permissions as the single upload route.
- Add a route-specific request body limit for
  `/api/v1/knowledge-bases/{knowledgeBaseId}/document-batches`.
- Proposed constants:
  - `knowledgeDocumentBatchMaxFiles = 10`
  - `knowledgeDocumentMaxFileBytes = 32 << 20`
  - `knowledgeDocumentBatchMultipartOverhead = 4 << 20`
  - `knowledgeDocumentBatchUploadMaxBodyBytes =
    knowledgeDocumentMaxFileBytes * knowledgeDocumentBatchMaxFiles +
    knowledgeDocumentBatchMultipartOverhead`

Gateway does not parse multipart fields and does not inspect file count.

### Knowledge Adapter

Changes:

- Add a route handler for `document-batches`.
- Parse multipart body with a batch max body limit derived from the same 10-file
  and 32 MiB per-file contract.
- Extract file headers from `r.MultipartForm.File["files"]`.
- Validate:
  - no files -> request-level validation error
  - more than 10 files -> request-level validation error
  - zero-byte file -> per-file failed result
  - file larger than 32 MiB -> per-file failed result
- For each valid file, infer content type with the existing
  `documentContentTypeFromFilename` fallback, so `.csv` maps to `text/csv`.
- Upload each valid file through the existing vendor client.
- If auto-ingestion is enabled and upload returns an ID, call
  `StartDocumentParse` for that document. Parse-start failure is a per-file
  failure. If parse fails after upload, attempt the same cleanup as current
  single-upload behavior for that file only, but do not roll back other files.
- Convert successful vendor documents through existing `documentFromVendor`.
- Convert expected vendor/application errors into sanitized per-file error codes
  using existing `mapVendorError` classification; avoid leaking raw vendor
  details.

Processing order is sequential for the first implementation. Sequential
processing is simpler, preserves runtime pressure, and keeps per-file error
handling deterministic. Parallelization can be a later performance task if real
usage needs it.

## Frontend Design

### API And Hooks

- Update Gateway OpenAPI and regenerate `apps/web/src/api/generated/gateway.ts`.
- Add `uploadDocumentBatch(knowledgeBaseId, files, tags)` in
  `apps/web/src/api/knowledge.ts`.
- Add `useUploadDocumentBatch()` in
  `apps/web/src/features/knowledge/hooks/use-documents.ts`.
- Invalidate document list and knowledge-base queries when `successCount > 0`.

### UI

Update `KnowledgeDocumentsPage` upload dialog:

- Store a list of selected upload items instead of one `selectedFile`.
- File input uses `multiple`.
- Drop handler iterates all dropped files.
- Client-side validation keeps valid files and reports rejected files.
- Submit one batch request with selected valid files.
- Render per-file result rows for success/failure.
- Leave failed items available for retry after a partial failure.
- Close/reset dialog only when all submitted files succeed.

Existing CSV extension/MIME lists remain, with tests asserting CSV acceptance.

## Compatibility

- Existing single-upload route remains unchanged for compatibility.
- Existing frontend code can migrate to the batch route while single-upload API
  tests continue to cover the old endpoint.
- The new route is additive in OpenAPI and Gateway routes.
- No database migration is required.
- No Knowledge runtime API change is required.

## Rollback

Reverting the feature means removing the additive batch route, schemas, frontend
batch wrapper/hook/UI changes, and regenerated type changes. The existing
single-upload route remains available throughout the implementation, so rollback
does not require data migration.

## Risks

- Large multipart requests can hit Gateway or Knowledge body limits. Mitigation:
  route-specific Gateway limit and Knowledge batch parser limit.
- Long sequential batches can approach request timeouts if all files are large
  or runtime is slow. Mitigation: 10-file first-slice limit and sequential,
  deterministic implementation; document any timeout issue as follow-up if
  observed.
- `207 Multi-Status` may need frontend client support if the wrapper treats any
  `2xx` as success. Verify `apps/web/src/api/client.ts` accepts all `2xx`
  statuses before implementation.
- OpenAPI generated type changes can ripple through frontend typecheck.
  Mitigation: regenerate once after contract changes and keep wrappers typed.
