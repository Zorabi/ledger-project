# Contributing

Thank you for helping improve Gift Ledger.

## Before you start

- Search existing issues before opening a new one.
- Use an issue to discuss substantial product, API, database, or architecture changes before implementation.
- Do not include real household data, credentials, exported workbooks, email addresses, or unredacted screenshots in issues or pull requests.

## Development workflow

1. Fork the repository and create a focused branch.
2. Keep changes scoped and follow the existing Go and Vue project structure.
3. Update documentation and `CHANGELOG.md` when behavior, APIs, configuration, deployment, or user-visible workflows change.
4. Run the relevant checks:

   ```bash
   cd backend && go test ./...
   cd frontend && npm run build
   ```

5. Open a pull request describing the problem, the solution, the verification performed, and any migration or compatibility impact.

## Project invariants

- Household data must be isolated by the authenticated user's `family_id`; never trust a family ID supplied by the frontend.
- Permission checks must use the fixed `super_admin`, `family_admin`, and `member` roles.
- User management, imports, backups, dictionaries, and important business mutations must be audited.
- Monetary values are stored as integer cents in PostgreSQL.
- Core business data uses soft deletion or disabling rather than physical deletion.
- API changes should keep Swagger metadata, request/response DTOs, and error contracts in sync.
- UI changes must remain practical on both desktop and mobile layouts.

## Commit style

Use a focused commit with the project format:

```text
{emoji} {type}({scope}): {Chinese description}
```

Examples include `✨ feat(stats): 增加统计维度` and `🐛 fix(auth): 修复会话失效判断`.

## License

By contributing, you agree that your contributions are licensed under the repository's [AGPL-3.0 license](LICENSE).
