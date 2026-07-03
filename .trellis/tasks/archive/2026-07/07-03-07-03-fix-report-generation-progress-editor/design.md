# Design

## Boundaries

- Frontend owns display semantics in
  `apps/web/src/pages/reports/generate/page.tsx` and report-specific query/API
  helpers under `apps/web/src/features/reports/`.
- Document service owns durable job progress and section generation semantics in
  `services/document/internal/service/report_generation_service.go`.
- No Gateway route or OpenAPI semantic change is required. The existing
  contract exposes `ReportJob.progress` as an object; Document/Trellis specs
  already define numeric `completed` and `total` as the stable progress fields.

## Frontend Data Flow

1. User creates a report and an outline/content job.
2. `useReportJobQuery` polls `/report-jobs/{jobId}`.
3. Page maps the latest job plus current report/template/type data into a
   user-facing progress view.
4. Progress mapping rules:
   - `succeeded` for content/report workflow -> 100%.
   - `partial_succeeded` -> 100% with partial-completion status text.
   - `failed` -> use attempted progress when available; otherwise keep a small
     failure percentage rather than claiming completion.
   - `progress.percent` remains a backward-compatible direct percentage.
   - `progress.completed` / `progress.total` drives real section progress.
   - outline running uses an elapsed-time optimistic percentage capped below the
     outline-complete portion, so users see that work is active even before the
     first backend progress write.
   - content running maps section completion into the remaining portion.
5. The model summary shown on this page is document-generation-specific. It
   reads the effective published document model settings/profile and does not
   render the QA/current LLM configuration card.
6. The cancel affordance stays visible for pending/running report jobs. The
   current frontend entrypoint still reports the explicit unsupported Gateway
   cancellation-contract error until a real cancellation route is reviewed and
   added.

## Backend Data Flow

`executeContentGeneration` already loops through target sections and updates
section status before and after AI calls. The current bug is early return after
section-level failures. The revised flow:

1. Sort target sections and initialize `processed`, `succeededOrSkipped`, and
   `failed` counters.
2. For each section:
   - skip preserved manual edits and count as processed;
   - if marking running fails, record failure and treat it as a fatal
     infrastructure error only for that section, then continue when safe;
   - if context/model/parse fails, mark the section failed, record
     `section.failed`, increment processed progress, and continue;
   - if stale/concurrent write conflict occurs, record `section.skipped`,
     increment processed progress, and continue;
   - if persistence fails in a way that indicates repository infrastructure
     failure, preserve existing rollback behavior and return dependency error.
3. At the end:
   - no failures -> `succeeded`;
   - failures plus at least one succeeded/skipped/processed section ->
     `partial_succeeded`;
   - all sections failed from provider/content failures -> return an error so
     worker marks the job failed, but progress still reflects all sections were
     attempted.

## Retry And Content Generation Ordering

- `partial_succeeded` is a terminal state whose work reached per-section
  outcomes. Manual retry should therefore use the same attempt resource as
  failed/canceled jobs and reset the job to `pending` for requeue.
- Report-level content generation remains sequential in this task. Section
  bodies can depend on preceding generated context and consistent report-wide
  phrasing, so bounded parallel workers are deferred until a separate reviewed
  context-sharing design exists.
- Progress uses completed terminal outcomes rather than section loop index, so
  failed sections and skipped sections both advance the workflow without
  corrupting `completed/total`.
- Running-marker failures remain fatal infrastructure failures and do not call
  the AI provider. Later follow-up work may relax that, but this task only makes
  provider/context/parse/persistence failures continue per section.

## UI Layout

- Remove the right-side current-report card.
- Remove the QA/current LLM configuration card from this page and keep the
  document model configuration module as the only model summary/control.
- Keep the cancel task button wired to the report cancel helper so the UI does
  not lose the existing cancel affordance while the backend contract is pending.
- In the content step, constrain the section list to a scrollable pane and make
  the editor column a stable-height flex area with a scrollable text editor.
  This keeps selection and body editing in the same viewport on desktop while
  preserving stacked layout on smaller screens.

## Compatibility

- Existing jobs that return only `percent` continue to render.
- Existing jobs that return `{ completed, total }` render accurate progress.
- Existing partial-success semantics remain available, but report-level content
  generation now attempts later sections before deciding final status.
