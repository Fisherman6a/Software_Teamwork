# Report Streaming Output

## Goal

Add real-time streaming output for report outline generation and report body generation so users can see text arrive while long-running report jobs are still executing.

## Background

- The report generation page currently creates asynchronous `ReportJob` resources and polls job, outline, section, and event resources.
- Document owns report generation business logic; Gateway must only expose and proxy browser-facing routes.
- Browser code may call only Gateway `/api/v1/**`.
- Existing QA streaming uses Gateway SSE and the frontend already has a reusable `fetch` stream reader.
- Document report events are already persisted and exposed through `GET /reports/{reportId}/events`.
- Report generation currently calls AI Gateway chat in non-streaming mode and persists only final outline / section results.
- Prior knowledge-retrieval work in the current dirty tree adds optional `knowledgeBaseIds`, retrieval degradation events, and section version knowledge sources. This task must preserve that behavior.

## Requirements

1. Outline generation must expose incremental text while the outline job is running.
2. Report body generation must expose incremental text while sections are being generated.
3. Streaming must be available only through Gateway public routes, not by direct frontend calls to Document or AI Gateway.
4. Gateway OpenAPI must document any new SSE route before the frontend relies on it.
5. Document must remain the owner of report event shape and report generation behavior.
6. AI provider streaming failures must fail or degrade according to existing generation semantics, but a browser SSE connection failure must not fail the report job.
7. Existing polling must remain as a fallback and recovery path.
8. The frontend stream must support cancellation through `AbortController` and must not leak prompts, provider raw errors, internal URLs, object keys, or full private tool payloads.
9. Knowledge retrieval degradation must continue to be non-fatal and visible as a warning.
10. The implementation must not rewrite or revert existing uncommitted knowledge-retrieval changes.

## Acceptance Criteria

- [ ] Gateway exposes `GET /api/v1/reports/{reportId}/events/stream` as `text/event-stream` with owner `document`.
- [ ] Document exposes `GET /reports/{reportId}/events/stream` and emits report events as SSE frames.
- [ ] Gateway marks the report event stream route as streaming and proxies it without normal fixed request timeout interruption.
- [ ] AI Gateway chat streaming is consumed by Document for outline and section generation when supported.
- [ ] Document records short streaming delta events for outline and section generation before final persisted outline / section results.
- [ ] Final outline and section persistence continues to use the accumulated model response and existing validation/parsing.
- [ ] The report generation page opens the report event stream during outline/content jobs and shows a concise live output preview.
- [ ] If the SSE connection errors, the page keeps polling and displays the stable job state without failing the job.
- [ ] Unit/contract tests cover Document SSE, AI Gateway streaming client parsing, generation delta recording, Gateway route/OpenAPI streaming contract, and frontend stream preview.

## Out Of Scope

- Replacing report polling completely.
- Persisting large streaming payloads or full provider chunks.
- Changing report job lifecycle statuses.
- Changing provider credentials or local AI Gateway configuration.
- Full end-to-end real-provider smoke; this task targets fake-backed unit/contract coverage unless the local stack is already available.

## Open Questions

None blocking. The user asked to start implementation now and requested progress notes under `.local/`.
