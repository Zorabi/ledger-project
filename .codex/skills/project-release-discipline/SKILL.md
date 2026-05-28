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
   - Use the project version format `X.A.B.C` exactly, for example `1.1.0.2`.
   - Interpret the segments as: `X` = major version for重大变更/不兼容变更, `A` = minor version for功能模块新增, `B` = iteration version for existing feature iteration, `C` = patch version for bug fixes, documentation corrections, release-rule fixes, and other small compatible changes.
   - Bump the application version for every commit according to the highest-impact change in that commit. Reset lower-order segments to `0` when bumping a higher-order segment.
   - If the user does not specify a version, choose the next version from the change type: incompatible change bumps `X`, new functional module bumps `A`, feature iteration bumps `B`, bug/doc/rule fix bumps `C`.
   - Create a brand-new `CHANGELOG.md` version section for every commit. Do not append a new commit's changes to an existing version section, even when multiple commits happen on the same calendar day.
   - Write each changelog version section as the repository diff between the current commit and its immediate parent commit. Do not record conversation decisions, intent negotiation, or broader work that is not present in that commit's diff.
   - For an initial release commit that creates the product from an empty or effectively empty baseline, classify delivered capabilities as `Added`; do not use `Changed` or `Fixed` for features that did not exist before that commit.
   - Place the newest version section at the top and use `## vX.A.B.C - YYYY-MM-DD`.
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
- Do not merge any later commit content into an older version section. Each commit maps to exactly one new changelog version section, even if the later commit only changes documentation or release discipline.
- Changelog content must describe only concrete file/API/UI/data/documentation changes introduced by that commit relative to its parent commit.
- Initial release sections should usually contain `Added` and optional `Documentation` only, because there is no prior shipped behavior to change or fix.
- Product requirements should describe expected behavior, not implementation trivia.
- Architecture docs should describe stable technical decisions, not one-off bug history.
- Do not update backend Swagger docs for frontend-only style changes.

## Final Response

When the task includes a commit, report:

- Changed areas.
- Verification commands and result.
- Commit hash and Chinese commit message.
