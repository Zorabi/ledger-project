# 礼尚往来

<p align="center">
  面向家庭场景的自托管礼金、礼品、关系与回礼管理账本。
</p>

<p align="center">
  <a href="README.md">English</a> · <a href="README_zh-CN.md">简体中文</a>
</p>

<p align="center">
  <img alt="版本" src="https://img.shields.io/badge/version-1.6.0.3-2f855a">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white">
  <img alt="Vue" src="https://img.shields.io/badge/Vue-3-42b883?logo=vuedotjs&logoColor=white">
  <img alt="PostgreSQL" src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white">
  <a href="LICENSE"><img alt="开源协议" src="https://img.shields.io/badge/license-AGPL--3.0-blue"></a>
</p>

“礼尚往来”用于记录家庭收到和送出的礼金、礼品及背后的关系信息。系统将快速录入、往来对象、回礼跟踪、统计分析、Excel 备份恢复、家庭租户隔离和轻量审计整合在一个兼容 Web 与手机端的应用中。

## 系统截图

以下截图中的姓名和家庭标识均已脱敏。

### 工作台与快速录入

![工作台与快速录入](docs/assets/screenshots/dashboard.png)

### 收送统计概览

![收送统计概览](docs/assets/screenshots/analytics.png)

## 核心能力

- **家庭租户隔离**：每个家庭拥有独立数据范围，业务查询从登录上下文获取 `family_id`，不信任前端传入的家庭 ID。
- **固定三级角色**：`super_admin`、`family_admin`、`member`，菜单、路由和接口按角色控制权限。
- **高效流水录入**：记录收礼或送礼、现金或礼品、事项、日期、经办人、备注和回礼状态。
- **对象与关系管理**：往来对象独立于用户账号维护，并通过系统或家庭数据字典分类。
- **回礼处理**：支持待回礼、已回礼、关联流水和无需回礼等状态，避免不透明的复杂自动匹配。
- **固定统计分析**：提供收送合计、趋势、成员归属、关系、事项、待回礼和无需回礼等分析。
- **可恢复备份**：导出家庭完整 Excel，支持定时邮件、可选加密 ZIP 和合并导入恢复。
- **运维与安全基础**：Access/Refresh 会话、首次登录强制改密、登录限流、结构化日志、软删除和审计留存。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go、Gin、GORM、Swagger |
| 前端 | Vue 3、Vite、Naive UI、ECharts |
| 数据库 | PostgreSQL |
| 数据便携 | Excel 导入导出、可选加密 ZIP |
| 部署 | Docker Compose，或独立部署前后端进程 |

```text
浏览器
  └─ Vue 3 单页应用
       └─ REST API（Go + Gin）
            ├─ PostgreSQL
            ├─ 进程内备份、会话与审计清理调度
            └─ 用于定时备份和导入前备份的 SMTP 服务
```

## 使用 Docker 快速启动

### 环境要求

- Docker 24 或更高版本
- Docker Compose v2

```bash
git clone https://github.com/Zorabi/ledger-project.git
cd ledger-project
docker compose up -d --build
```

容器健康后可访问：

| 服务 | 地址 |
| --- | --- |
| Web 应用 | <http://localhost> |
| 后端 API | <http://localhost:8080> |
| Swagger UI | <http://localhost:8080/swagger/index.html> |
| PostgreSQL | `127.0.0.1:5432/ledger` |

开发环境初始化账号为 `admin` / `ChangeMe123!`，首次登录后请立即修改密码。Docker 与 YAML 中的默认凭据仅用于本地体验，禁止直接用于生产环境。

常用命令：

```bash
docker compose ps
docker compose logs -f
docker compose down
```

## 本地开发

### 环境要求

- Go 1.24+
- Node.js 20+ 和 npm
- PostgreSQL 16+，也可以使用 Docker 启动数据库

先启动 PostgreSQL，再分别启动后端和前端：

```bash
docker compose up -d postgres
make dev-backend
make dev-frontend
```

Vite 开发地址为 <http://localhost:5173>。

运行项目验证：

```bash
make test
```

构建当前平台发布产物：

```bash
make build
```

运行 `make help` 可查看跨平台后端构建、前端打包、SQL 迁移、数据库备份和恢复等命令。

## 配置说明

后端默认读取 `backend/config.yaml`，并支持环境变量覆盖。请以 `.env.example` 为配置参考，不要提交已经填写生产凭据的配置文件。

| 环境变量 | 用途 |
| --- | --- |
| `DATABASE_POSTGRES_*` | PostgreSQL 连接以及自动建库、迁移开关 |
| `JWT_SECRET` | Token 签名密钥，生产环境必须使用足够长的随机值 |
| `APP_ENCRYPTION_KEY` | 加密保存备份 ZIP 密码的服务端主密钥 |
| `APP_VERSION` | 系统展示版本号 |
| `EMAIL_*` | 发送备份邮件所使用的 SMTP 配置 |
| `AUDIT_RETENTION_DAYS` | 审计日志留存天数：`30`、`60` 或 `90` |
| `LOG_*` | 结构化控制台/文件日志及轮转配置 |

生产部署必须替换所有示例密码和密钥；手动管理数据库版本时应关闭自动建库与自动迁移；按部署环境启用 TLS；迁移服务器时必须保留 `APP_ENCRYPTION_KEY`。

## 备份与恢复安全规则

- 备份文件是单个家庭的完整 Excel 工作簿。
- 定时备份可通过邮件发送，并可选用加密 ZIP 附件。
- 恢复采用合并导入，不会先清空家庭数据。
- 正式导入前必须邮件发送当前家庭备份；邮件发送失败时中止导入。
- 导入文件和邮件附件可能包含个人与财务敏感信息，应按敏感数据要求存储和传输。

## 生产部署与数据库迁移

- 生产环境推荐共享 PostgreSQL 实例，但为本系统分配独立 database。
- 建议关闭 `DATABASE_POSTGRES_AUTO_CREATE` 和 `DATABASE_POSTGRES_AUTO_MIGRATE`，按 `sql/<version>/` 目录从小到大执行迁移。
- 每个版本目录先执行 `001_ddl.sql`，再执行 `002_data.sql`。
- 生产数据库无需暴露给开发机，可在应用服务器、堡垒机、CI/CD 内网任务、DBA 工作台或 SSH 隧道环境中使用 `psql`。
- 升级前使用 `pg_dump` 备份；迁移服务器时使用 `pg_restore` 恢复数据库，并同步密钥和运行配置。

示例：

```bash
pg_dump --format=custom --file=backups/ledger.dump \
  'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require'

psql 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require' \
  -v ON_ERROR_STOP=1 -f sql/<version>/001_ddl.sql
psql 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require' \
  -v ON_ERROR_STOP=1 -f sql/<version>/002_data.sql
```

完整迁移约束请参阅 [SQL 版本目录说明](sql/README.md) 和 [架构文档](docs/architecture.md)。

## 项目文档

- [English README](README.md)
- [产品需求](docs/product-requirements.md)
- [架构设计](docs/architecture.md)
- [数据库迁移说明](sql/README.md)
- [更新日志](CHANGELOG.md)
- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)

## 参与贡献

欢迎提交 Issue 和 Pull Request。提交前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。修改业务功能时，请同时检查家庭租户隔离、认证鉴权、审计日志、接口兼容性以及 Web/手机端响应式体验。

## 开源协议

Copyright © 2026 礼尚往来项目贡献者。

本项目采用 [GNU Affero General Public License v3.0](LICENSE) 开源。若您修改本软件并通过网络向用户提供服务，AGPL 要求向这些用户提供相应源代码。完整条款以 `LICENSE` 文件为准。
