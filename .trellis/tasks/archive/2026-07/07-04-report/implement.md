# Report Streaming Output Implementation Plan

## Execution Rules

- Use TDD for production behavior changes: write failing tests first, run them, then implement.
- Preserve existing uncommitted knowledge-retrieval changes.
- Do not manually edit generated frontend API types except through `bun run --cwd apps/web api:generate`.
- Keep `.local/report-streaming-progress.md` updated after each major step.

## Ordered Checklist

1. Activate this Trellis task with `python ./.trellis/scripts/task.py start .trellis/tasks/07-04-report`.
2. Load Phase 2.1 detail with `python ./.trellis/scripts/get_context.py --mode phase --step 2.1 --platform codex`.
3. Add failing Document HTTP test for `GET /reports/{reportId}/events/stream` returning SSE and emitting a persisted report event.
4. Implement Document SSE handler and route.
5. Add failing AI Gateway client test for streaming chat request headers/body and delta parsing.
6. Implement `StreamChatCompletion` in the Document AI Gateway client.
7. Add failing ReportGenerationService tests proving outline and section generation record delta events before final success.
8. Implement generation streaming with final accumulated response parsing.
9. Add Gateway OpenAPI route and failing Gateway route/contract tests for report SSE.
10. Implement Gateway route matrix and streaming timeout/proxy updates.
11. Regenerate frontend Gateway types with `bun run --cwd apps/web api:generate`.
12. Add failing frontend tests for report event stream parsing and live preview rendering.
13. Implement report stream hook and page preview UI.
14. Run targeted backend tests after each backend slice.
15. Run targeted frontend tests after the frontend slice.
16. Run final checks or record blockers:
    - `cd services/document && go test ./...`
    - `cd services/document && go build ./cmd/server`
    - `cd services/gateway && go test ./...`
    - `cd services/gateway && go build ./cmd/server`
    - `bun run --cwd apps/web check`
    - `bun run --cwd apps/web build`
    - `git diff --check`

## Risk Points

- `report_events` currently stores only a short `message`; delta payloads must stay compact.
- `ListReportEventsByReportID` ordering may be newest-first; SSE handler must emit chronological order and avoid duplicates.
- Stream UI must not overwrite final user-editable content with partial preview text.
- Browser stream failure must not be shown as generation failure.
- Existing knowledge retrieval degradation and traceability changes must remain intact.

## Manual Inspection Notes

After implementation, the user should inspect:

- Gateway OpenAPI diff for the new report SSE route.
- Report generation page during outline generation and content generation.
- Knowledge-retrieval warning behavior when retrieval fails.
- Final outline/sections after a streamed generation completes.
