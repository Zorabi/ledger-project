---
name: project-release-discipline
description: Project-specific release hygiene for this ledger project. Use when Codex is asked to commit code, prepare a delivery, update changelog/docs, change requirements or architecture, or modify user-facing behavior. Enforces Chinese commit messages, documentation synchronization, changelog updates, and pre-commit verification.
---

# Project Release Discipline

## Workflow

Use this workflow before committing or presenting a completed delivery in this repository.

1. Inspect the worktree with `git status --short`.
   - If there are no changes, tell the user there is nothing to commit and stop.
   - If there are changes, review the actual diff before deciding the version and changelog content.
2. Read the latest version from the top version section in root `CHANGELOG.md`.
3. Decide the next version from the highest-impact change in the current diff:
   - Major / incompatible change: increment `X`, reset `A.B.C` to `0.0.0`.
   - New functional module: increment `A`, reset `B.C` to `0.0`.
   - Existing feature iteration or refactor: increment `B`, reset `C` to `0`.
   - Bug fix, documentation, test, build/config, release-rule fix: increment `C`.
4. Show the user the suggested next version and the reason for the increment, then wait for user confirmation before changing version files or committing.
5. Update the relevant documentation after confirmation:
   - `CHANGELOG.md` for every user-visible, API, data, UI, deployment, or workflow change.
   - `docs/product-requirements.md` for changed product behavior, interaction rules, or user-facing constraints.
   - `docs/architecture.md` for changed technical architecture, API contracts, persistence, responsive strategy, or frontend structure.
   - `backend/docs/*` when backend API contracts change.
   - `sql/<version>/001_ddl.sql` and `sql/<version>/002_data.sql` when database tables, indexes, fields, or preset data change.
   - `README.md` when startup, deployment, database migration, backup, restore, or server migration behavior changes. Production database commands must not assume direct access from the developer machine; prefer native `psql`, `pg_dump`, and `pg_restore` examples, and document execution from a reachable host, bastion, CI/CD job, DBA console, or SSH tunnel when relevant. Do not make `Makefile` a production dependency.
6. Update the application version for every code commit:
   - Use the project version format `X.A.B.C` exactly, for example `1.1.0.2`.
   - Interpret the segments as: `X` = major version for重大变更/不兼容变更, `A` = minor version for功能模块新增, `B` = iteration version for existing feature iteration, `C` = patch version for bug fixes, documentation corrections, release-rule fixes, and other small compatible changes.
   - Bump the application version according to the confirmed version.
   - Create a brand-new `CHANGELOG.md` version section for every commit. Do not append a new commit's changes to an existing version section, even when multiple commits happen on the same calendar day.
   - Write each changelog version section as the repository diff between the current commit and its immediate parent commit. Do not record conversation decisions, intent negotiation, or broader work that is not present in that commit's diff.
   - For an initial release commit that creates the product from an empty or effectively empty baseline, classify delivered capabilities as `Added`; do not use `Changed` or `Fixed` for features that did not exist before that commit.
   - Place the newest version section at the top and use `## vX.A.B.C - YYYY-MM-DD`.
   - Keep the product version synchronized across `backend/config.yaml`, `backend/internal/config/config.go`, `backend/cmd/api/main.go`, `backend/docs/*`, `frontend/src/App.vue`, and `CHANGELOG.md`.
   - Keep frontend package-manager versions valid SemVer in `frontend/package.json` and `frontend/package-lock.json`; use `X.A.B` there when the product version is `X.A.B.C`.
7. Run focused verification. Prefer the project’s existing commands:
   - Frontend-only changes: `npm run build` in `frontend`.
   - Backend/API changes: `go test ./...` in `backend`.
   - Full-stack changes: run both.
8. Stage intentionally and inspect the staged diff or `git status --short`.
9. Commit with a concise Chinese commit message using the required format below.

## Commit Message Rules

- Commit message format: `{emoji} {type}({scope}): {description}`.
- Use Chinese for `{description}`.
- Emoji and type mapping:
  - `✨ feat`: 新功能
  - `🐛 fix`: Bug 修复
  - `♻️ refactor`: 重构
  - `📝 docs`: 文档
  - `✅ test`: 测试
  - `🔧 chore`: 配置/构建
- Choose a concise scope such as `deploy`, `db`, `ui`, `auth`, `docs`, `release`, or `backend`.
- Example: `✨ feat(deploy): 增加生产部署和数据库迁移说明`
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
