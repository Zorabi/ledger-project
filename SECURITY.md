# Security Policy

Gift Ledger stores personal relationship and financial records. Please handle vulnerability reports and sample data carefully.

## Supported versions

Security fixes are applied to the latest released version. Upgrade to the newest release before reporting an issue that may already be resolved.

## Reporting a vulnerability

Please use GitHub's **Private vulnerability reporting** or open a private security advisory for this repository. Do not disclose exploitable details in a public issue.

Include, when possible:

- affected version and deployment model;
- impact and prerequisites;
- minimal reproduction steps;
- relevant request/response details with tokens, names, email addresses, and household data removed;
- a suggested mitigation, if known.

## Deployment responsibilities

- Replace every sample password, JWT secret, and encryption key before production use.
- Keep `APP_ENCRYPTION_KEY` available during server migration and out of source control.
- Protect database backups, Excel exports, encrypted ZIP passwords, SMTP credentials, and logs as sensitive data.
- Use TLS for public deployments and restrict direct PostgreSQL access.
- Review audit logs and keep dependencies and container images current.

Please do not upload production configuration files, database dumps, backup workbooks, or unredacted screenshots with a report.
