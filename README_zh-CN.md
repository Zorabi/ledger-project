# 礼尚往来

用于记录家庭礼金、礼品往来，管理往来对象、回礼状态和家庭备份的自托管账本。

![版本](https://img.shields.io/badge/version-1.6.0.5-2f855a)
[![许可证：AGPL-3.0-only](https://img.shields.io/badge/license-AGPL--3.0--only-blue)](LICENSE)

## 功能

- **往来记录**：记录收礼、送礼、金额、礼品、事项、日期、经办人和备注，支持查询与编辑。
- **对象管理**：维护往来对象、关系和分类，用户账号与往来对象独立管理。
- **回礼管理**：手动维护回礼状态，可关联对应流水。
- **统计分析**：查看收送合计、月度趋势、成员归属及待回礼等统计。
- **家庭协作**：按家庭隔离数据，提供系统管理员、家庭管理员和成员三种固定角色。
- **备份恢复**：导出家庭完整 Excel，支持定时邮件备份、可选加密 ZIP 和合并导入。
- **操作追踪**：记录重要操作的审计日志，核心业务数据采用软删除或禁用。

## 效果预览

工作台将收送汇总、快速录入和最近流水放在同一页面，便于日常记账。截图使用脱敏示例数据，界面版本仅作展示。

![工作台：收送汇总、快速录入和最近流水](docs/assets/screenshots/dashboard.png)

<details>
<summary>统计页面：月度趋势、成员归属与回礼状态</summary>

![统计页面：月度收送趋势、归属成员统计与回礼状态](docs/assets/screenshots/analytics.png)

</details>

## 安装与快速开始

项目使用 Go 后端、Vue 3 前端和 PostgreSQL。以下步骤适用于本地体验与开发；需要 Git、Go 1.24+、Node.js（建议使用与前端构建镜像一致的 22 版本）、npm，以及 Docker Compose v2。

1. 获取源码并启动数据库：

   ```bash
   git clone https://github.com/Zorabi/ledger-project.git
   cd ledger-project
   docker compose up -d postgres
   docker compose ps
   ```

   等待 PostgreSQL 状态为 `healthy`。默认占用本机 `5432` 端口；如已有数据库，可直接在后端配置中填写连接信息。

2. 检查 `backend/config.yaml`，确认数据库连接、管理员初始账号及密钥配置。仓库默认数据库为 `ledger`，用户为 `postgres`，密码为 `123456`；首次启动默认开启自动建库和迁移，仅适合本地体验。

3. 在仓库根目录启动后端：

   ```bash
   cd backend
   go mod download
   go run ./cmd/api
   ```

4. 新开终端，从仓库根目录安装依赖并启动前端：

   ```bash
   cd frontend
   npm ci
   npm run dev
   ```

5. 打开 <http://localhost:5173>。新数据库默认初始化系统管理员 `admin`，初始密码为 `ChangeMe123!`；首次登录按提示修改密码。

| 服务 | 本地地址 |
| --- | --- |
| Web 界面 | <http://localhost:5173> |
| 后端 API | <http://localhost:8080> |
| Swagger 接口文档 | <http://localhost:8080/swagger/index.html> |

Vite 已将 `/api` 请求代理到本地后端。仓库也提供完整的 Compose 服务定义，但前端容器使用默认 Nginx 配置；用于完整应用部署前，还需为 `/api` 配置指向后端的反向代理。

停止体验用数据库时，在仓库根目录运行 `docker compose stop postgres`，数据保存在 Docker 卷中。

## 使用说明

1. 系统管理员创建家庭及家庭管理员；普通用户不能自行创建家庭。
2. 家庭管理员维护本家庭成员、自定义字典和备份配置。
3. 先添加往来对象，再通过工作台或往来流水页面录入收礼、送礼记录。记录保存后直接生效。
4. 按实际往来情况手动更新回礼状态，需要时关联另一条流水；在统计页面查看家庭收送情况。
5. 在备份配置中设置收件人和频率，或导出家庭 Excel。恢复时使用同格式文件合并导入，不会先清空现有数据。

正式导入前，系统必须通过邮件发送当前家庭备份；SMTP 未配置或发送失败时会中止导入。导出文件和邮件附件含家庭业务数据，请妥善保存。

## 配置与部署

后端读取 `backend/config.yaml`，支持同名环境变量覆盖（例如 `JWT_SECRET`）。[环境变量示例](.env.example)可作为字段参考；后端不会自动加载 `.env` 文件，本地运行时需由启动环境显式传入。

| 配置 | 用途 |
| --- | --- |
| `DATABASE_POSTGRES_*` | 数据库连接、自动建库与迁移开关 |
| `JWT_SECRET` | 登录令牌签名密钥 |
| `APP_ENCRYPTION_KEY` | 加密保存备份 ZIP 密码的主密钥 |
| `EMAIL_*` | 备份邮件 SMTP 配置 |
| `AUDIT_RETENTION_DAYS` | 审计留存天数，支持 `30`、`60`、`90` |
| `LOG_*` | 日志输出和轮转配置 |

部署前替换示例账号密码和密钥，配置 HTTPS，并持久化数据库。使用独立数据库迁移流程时关闭自动建库和自动迁移。数据库维护步骤见 [数据库部署与迁移说明](sql/README.md)；配置和组件关系见 [架构文档](docs/architecture.md)。

## 兼容性

| 项目 | 当前范围 |
| --- | --- |
| 客户端 | 浏览器访问，响应式布局适配桌面与手机，界面为中文 |
| 数据库 | PostgreSQL；仓库 Compose 使用 PostgreSQL 16 |
| 金额 | 人民币，数据库以整数分存储 |
| 登录 | 账号密码，支持首次登录强制修改密码 |
| 附件 | 仅预留模型和扩展点，尚未开放上传 |

## 开发与测试

后端代码位于 `backend/`，前端代码位于 `frontend/`。安装前端依赖后，在仓库根目录运行：

```bash
make test
```

该命令依次运行 Go 测试和前端生产构建；前端构建不等同于端到端测试。也可分别执行：

```bash
(cd backend && go test ./...)
(cd frontend && npm run build)
```

构建当前平台后端二进制及前端静态资源压缩包：

```bash
make build
```

产物分别写入 `dist/backend/<系统>-<架构>/` 和 `dist/frontend/ledger-web.zip`。其他构建命令见 `make help`。

## 贡献与文档

提交 Issue 或 Pull Request 前请阅读 [贡献指南](CONTRIBUTING.md)。修改时请关注家庭数据隔离、权限、审计和移动端体验；不要在问题报告中附带真实家庭数据或凭据。安全问题请按 [安全策略](SECURITY.md)报告。

- [产品需求](docs/product-requirements.md)
- [架构设计](docs/architecture.md)
- [数据库部署与迁移](sql/README.md)
- [更新日志](CHANGELOG.md)

## 许可证

Copyright (c) 2026 dongx。

本项目基于 [GNU Affero General Public License v3.0](LICENSE) 发布。SPDX 标识：`AGPL-3.0-only`。
