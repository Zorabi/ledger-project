---
name: project-release-discipline
description: Project-specific release hygiene for this ledger project. Use when Codex is asked to commit code, prepare a delivery, update changelog/docs, change requirements or architecture, or modify user-facing behavior. Enforces Chinese commit messages, documentation synchronization, changelog updates, and pre-commit verification.
---

# Project Release Discipline

## Workflow

Use this workflow before committing or presenting a completed delivery in this repository.

1. Inspect the worktree with `git status --short` and review the actual diff.
2. Decide whether the change affects product behavior, architecture, API, data model, UI/UX, deployment, or developer workflow.
3. Update the relevant documentation:
   - `CHANGELOG.md` for every user-visible, API, data, UI, deployment, or workflow change.
   - `docs/product-requirements.md` for changed product behavior, interaction rules, or user-facing constraints.
   - `docs/architecture.md` for changed technical architecture, API contracts, persistence, responsive strategy, or frontend structure.
   - `backend/docs/*` when backend API contracts change.
4. Update the application version for every code commit:
   - Use the project version format `X.A.B.C` exactly, for example `1.0.1.0`.
   - Bump the version whenever committing code, even for UI-only changes.
   - Keep the version synchronized across backend config defaults, runtime config, frontend fallback display version, Swagger metadata, and the `CHANGELOG.md` version section.
   - Keep package-manager versions valid for their ecosystems if they cannot use four numeric segments; the product/application version remains the four-segment value exposed by `/api/v1/version`.
5. Run focused verification. Prefer the project’s existing commands:
   - Frontend-only changes: `npm run build` in `frontend`.
   - Backend/API changes: `go test ./...` in `backend`.
   - Full-stack changes: run both.
6. Stage intentionally and inspect the staged diff or `git status --short`.
7. Commit with a concise Chinese commit message. Do not use English-only commit messages.

## Commit Message Rules

- Use Chinese by default, for example:
  - `feat: 完善移动端视觉体验`
  - `fix: 修复流水日期提交异常`
  - `docs: 更新产品需求和架构文档`
- Keep the type prefix conventional when useful: `feat`、`fix`、`docs`、`style`、`refactor`、`test`、`chore`.
- The summary after the prefix must be Chinese.
- If amending an existing commit, preserve the intent but translate the message into Chinese.

## Documentation Rules

- Changelog entries should be professional and grouped by `Added`、`Changed`、`Fixed`、`Documentation` when possible.
- Changelog top-level version headings must include the four-segment version and release date, for example `## v1.0.1.0 - 2026-05-28`.
- Do not merge unrelated version content into an older version section; add a new version section when a later commit changes product behavior.
- Product requirements should describe expected behavior, not implementation trivia.
- Architecture docs should describe stable technical decisions, not one-off bug history.
- Do not update backend Swagger docs for frontend-only style changes.

## Final Response

When the task includes a commit, report:

- Changed areas.
- Verification commands and result.
- Commit hash and Chinese commit message.
