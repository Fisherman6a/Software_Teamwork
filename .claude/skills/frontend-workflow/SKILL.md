---
name: frontend-workflow
description: "Use when working on this project's frontend app, apps/web, frontend-dev branch workflow, fork/upstream PR flow, Bun frontend commands, frontend lint/build checks, or frontend collaboration rules."
---

# Frontend Workflow

Use this project-local skill before frontend implementation, review, branch/PR guidance, or CI/Lint setup for `apps/web`.

## Read First

1. `docs/frontend-workflow.md` — human-facing collaboration workflow.
2. `.trellis/spec/frontend/index.md` — agent-facing frontend implementation standards.
3. `.trellis/spec/frontend/quality-guidelines.md` — required checks, Bun command form, Git workflow, and review checklist.
4. `.trellis/spec/frontend/directory-structure.md` — `apps/web/src/` layout and module boundaries.

## Fixed Project Rules

- Frontend source lives under `apps/web/src/`; do not create repository-root `src/` for frontend code.
- The frontend integration branch is `upstream/frontend-dev`; do not use `frontdev` in commands, docs, CI, or PR targets.
- Create frontend feature branches from `upstream/frontend-dev`, push them to the developer fork `origin`, and open PRs back to `Sakayori-Iroha-168/Software_Teamwork:frontend-dev`.
- Use Bun for frontend dependencies and scripts.
- From the repository root, run frontend scripts as `bun run --cwd apps/web <script>`.

## Required Checks

For non-trivial frontend changes, run and report:

```bash
bun run --cwd apps/web check
bun run --cwd apps/web build
git diff --check
```

If a check is not run, report it with the reason and remaining risk.

## Trellis Fit

This skill is for on-demand frontend workflow context. Do not replace the Trellis workflow-state hooks with this content. Hooks should stay focused on active-task and phase context; frontend rules belong here and in `.trellis/spec/frontend/`.
