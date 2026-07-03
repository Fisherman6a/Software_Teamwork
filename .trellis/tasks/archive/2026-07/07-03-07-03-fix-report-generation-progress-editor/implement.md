# Implementation Plan

## Checklist

1. Add or update frontend tests for:
   - progress panel hides IDs/task type and shows template type;
   - `completed/total` maps to visible progress;
   - cancel task button is visible for pending/running jobs and current report
     card is absent;
   - long section list/editor layout has stable scroll containers.
2. Add or update Document service tests for:
   - content generation continues after a middle section fails;
   - content generation continues after a middle section persistence/version
     write failure;
   - progress reaches total attempted sections;
   - final status is `partial_succeeded` when some sections fail after other
     sections are processed.
   - retrying `partial_succeeded` jobs creates a new attempt and enqueues work.
   - content generation remains sequential while still advancing progress after
     each section terminal outcome.
3. Update `ReportGeneratePage`:
   - keep cancel import/hook/handler/button wired to the existing
     unsupported-contract helper until Gateway exposes real cancellation;
   - remove internal ID/task type fields;
   - remove right-side current-report module;
   - remove the QA/current LLM configuration card from this page;
   - show the current effective document generation model summary;
   - add template type summary;
   - replace progress calculation with a helper that reads `completed/total`,
     keeps `percent` fallback, and adds optimistic outline-running progress.
4. Update content editor layout classes so the list and editor stay aligned and
   scroll independently where needed.
5. Update Document content-generation loop to continue after section-level
   provider/context/parse failures and update progress after every attempted
   section.
6. Keep report-level content generation sequential, but make every terminal
   section outcome update progress so failed sections do not leave later
   sections stuck in `pending`.
7. Run targeted checks, then broader checks:
   - `bun run --cwd apps/web test:unit -- src/pages/reports/generate/page.test.tsx`
   - `bun run --cwd apps/web check`
   - `cd services/document && go test ./internal/service -run ReportGeneration`
   - `cd services/document && go test ./...`
   - `cd services/document && go build ./cmd/server`
   - `git diff --check`

## Risk Points

- Keep admin-only document model settings guard from PR #535, but do not show
  the QA/current LLM configuration card on the report generation page.
- Do not expose provider raw errors or prompts when changing section failure
  behavior.
- Do not make frontend progress claim full success when backend status is
  failed; distinguish completed processing from successful generation.
