# Fix report generation progress and editor UX

## Goal

Make the report generation flow understandable and reliable for report writers:
hide internal implementation IDs, show only useful progress and template context,
keep the current section editor visible, and let report content generation keep
processing later sections even when one section fails.

## Background And Evidence

- The report generation page currently exposes `reportId`, `jobId`, and raw
  `jobType` in the top progress panel and repeats report metadata in a separate
  right-side "当前报告" module. These fields are implementation details and not
  useful to normal users.
- The visible cancel button calls a frontend stub:
  `apps/web/src/features/reports/report-generation.api.ts` throws
  "Job cancellation is not yet supported by the Gateway contract." The user
  asked to keep this cancel affordance visible while the real Gateway contract
  is still pending.
- Backend job progress is stored and returned as numeric
  `progress.completed` / `progress.total`, while the frontend progress helper
  only reads `progress.percent`; real progress can therefore stay at 0 even when
  the worker updates completed sections.
- `services/document/internal/service/report_generation_service.go` currently
  returns early after a section generation failure once any earlier section has
  succeeded. Later sections are never attempted and remain `pending`, matching
  the observed "等待中不动" behavior.
- The content step uses a two-column grid with a growing section list and a
  fixed-height text area. With many sections, clicking a lower list item can
  leave the editor outside the visible viewport.

## Requirements

- R1: In the main report progress area, remove visible `reportId`, `jobId`, and
  raw task type. Keep user-facing status and a progress bar.
- R2: Add a user-facing "报告模板类型" line to the progress area. It should use
  the selected report type display name when available, falling back to the
  selected template/report type code only when necessary.
- R3: Remove the redundant right-side "当前报告" module.
- R4: Keep the cancel task button and `cancelReportJob` entrypoint on the report
  generation page. Until a real Gateway cancellation contract exists, clicking
  the button must surface the explicit unsupported-contract error instead of
  silently doing nothing.
- R5: Compute frontend progress from `ReportJob.progress.completed` and
  `ReportJob.progress.total`, while preserving backward compatibility with
  `progress.percent` if present.
- R6: Present progress as a single report workflow:
  - outline generation occupies an initial small portion of progress and should
    move slowly while running;
  - once outline generation succeeds, the workflow should visibly advance;
  - content generation uses completed/total section progress;
  - when all sections have been attempted, the progress reaches 100 and the UI
    shows an appropriate completion/partial-completion message.
- R7: During report-level content generation, a failed section must be marked
  failed and counted as processed, but generation should continue with later
  sections. The job should finish `partial_succeeded` when at least one section
  failed and at least one section was processed successfully or skipped.
- R8: The section list and section editor must use stable, aligned heights so
  selecting lower sections does not make the body editor disappear from view.
- R9: The report generation page must not show the QA/current LLM configuration
  card. It should show the currently effective document generation model
  instead, using the published document model settings/profile.
- R10: A `partial_succeeded` report generation job must be retryable through
  the existing `POST /report-jobs/{jobId}/attempts` resource creation flow so
  the retry button does not surface a validation error after partial content
  failures.
- R11: Section-level persistence failures, including generated section-version
  creation failures after the section update rolls back, must mark only that
  section failed, count it as attempted progress, and continue later sections.
- R12: Section content generation remains sequential in this task because later
  sections may depend on earlier generated context. Parallel workers are
  explicitly deferred until there is a reviewed context-sharing design.

## Acceptance Criteria

- [ ] The progress panel no longer renders `reportId`, `jobId`, or raw job type.
- [ ] The progress panel renders status, progress, and "报告模板类型".
- [ ] The right-side "当前报告" module is removed.
- [ ] A visible cancel-task button remains available for pending/running report
  jobs and reports the current unsupported Gateway cancellation contract when
  clicked.
- [ ] Frontend unit tests cover `completed/total` progress mapping and no-ID UI.
- [ ] Content generation service tests cover one failed middle section followed
  by a later attempted section.
- [ ] Failed sections no longer leave later sections stuck in `pending` solely
  because an earlier section failed.
- [ ] Section list/editor layout keeps the editor visible when selecting lower
  sections in a long list.
- [ ] The page shows "当前文档生成模型" and no longer shows the separate
  "当前 LLM 配置" card.
- [ ] Existing report generation page tests and Document service tests pass.
- [ ] Retrying a `partial_succeeded` job creates a new attempt and re-enqueues
  the job instead of returning request validation failed.
- [ ] A generated section-version persistence failure no longer leaves later
  sections stuck in `pending`.
- [ ] Section content generation remains sequential and reports progress once
  for each terminal section outcome.
