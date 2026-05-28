# 礼尚往来项目架构设计

## 项目结构

项目采用前后端分离结构：

```text
ledger-project/
  backend/    Go + Gin + GORM 后端
  frontend/   Vue 3 + Vite + Naive UI 前端
  docs/       产品和架构设计文档
```

后端使用 PostgreSQL。前端使用单应用模式，根据用户角色控制菜单、路由和页面权限。

## 后端模块建议

后端继续使用分层结构，并可按业务域逐步重构：

- `cmd/api`：应用入口、路由注册、服务初始化。
- `internal/config`：配置加载，支持环境变量覆盖。
- `internal/database`：PostgreSQL 连接、迁移入口。
- `internal/models`：GORM 模型。
- `internal/repository`：数据访问。
- `internal/services`：业务逻辑。
- `internal/handlers`：HTTP API。
- `internal/middleware`：认证、授权、日志、恢复。
- `internal/scheduler`：定时任务，包括备份任务和 session 清理。
- `internal/audit`：审计日志封装。
- `internal/exporter`：Excel 导出、ZIP 加密。
- `internal/importer`：Excel dry-run 和导入恢复。
- `internal/mailer`：邮件发送。

## PostgreSQL 数据模型草案

### 租户与用户

`families`

- `id`
- `name`
- `contact_name`
- `contact_email`
- `remark`
- `status`
- `admin_user_id`
- `created_at`
- `updated_at`
- `deleted_at`

`users`

- `id`
- `family_id`
- `username`
- `display_name`
- `email`
- `password_hash`
- `role`
- `must_change_password`
- `status`
- `last_login_at`
- `created_at`
- `updated_at`
- `deleted_at`

`user_sessions`

- `id`
- `user_id`
- `refresh_token_hash`
- `expires_at`
- `revoked_at`
- `ip`
- `user_agent`
- `created_at`

清理规则：每天清理过期 session 和已撤销超过 7 天的 session。

`auth_throttles`

- `id`
- `scope`：封禁维度，当前支持 `username` 和 `ip`
- `key`：规范化后的账号或访问 IP
- `failed_count`
- `window_started_at`
- `last_failed_at`
- `locked_until`
- `created_at`
- `updated_at`

登录防护规则：后端在登录失败时同时记录账号和访问 IP 维度；默认 15 分钟窗口内失败 5 次后临时封禁 30 分钟。管理员重置密码或登录成功后清理对应账号失败记录，后台定时清理过旧的失败/封禁记录。

### 数据字典

`dictionary_types`

- `id`
- `code`
- `name`
- `description`
- `allow_family_custom`
- `sort_order`
- `enabled`
- `created_at`
- `updated_at`

`dictionary_items`

- `id`
- `type_id`
- `family_id`
- `label`
- `value`
- `sort_order`
- `enabled`
- `is_system`
- `created_at`
- `updated_at`
- `deleted_at`

系统预置项 `family_id` 为空，家庭自定义项写入具体 `family_id`。

### 往来对象与记录

`contacts`

- `id`
- `family_id`
- `name`
- `contact_type`
- `relation_type`
- `phone`
- `address`
- `remark`
- `status`
- `created_by`
- `created_at`
- `updated_at`
- `deleted_at`

`gift_records`

- `id`
- `family_id`
- `contact_id`
- `occasion_type`
- `direction`
- `amount_cents`
- `gift_description`
- `occurred_on`
- `handled_by`
- `reciprocity_status`
- `related_record_id`
- `remark`
- `created_by`
- `created_at`
- `updated_at`
- `deleted_at`

回礼关联约束：

- `related_record_id` 仅能指向同家庭、同对象、相反方向的流水。
- 建立关联时，双方记录均写入对方 ID，并将 `reciprocity_status` 更新为 `done`。
- 取消关联、删除关联记录或更换关联对象时，被关联记录清空关联 ID，并回到 `pending`。

`attachments`

- `id`
- `family_id`
- `record_id`
- `file_name`
- `storage_key`
- `file_type`
- `file_size`
- `uploaded_by`
- `created_at`
- `deleted_at`

第一版预留表和接口边界，可不开放上传入口。

### 备份与导入

`backup_configs`

- `id`
- `family_id`
- `enabled`
- `frequency`
- `recipient_emails`
- `encrypt_enabled`
- `zip_password_encrypted`
- `zip_password_hint`
- `last_sent_at`
- `last_status`
- `created_at`
- `updated_at`

`backup_jobs`

- `id`
- `family_id`
- `trigger_type`
- `status`
- `file_name`
- `file_size`
- `file_sha256`
- `recipient_emails`
- `started_at`
- `finished_at`
- `error_message`
- `created_by`

`import_jobs`

- `id`
- `family_id`
- `operator_id`
- `status`
- `dry_run`
- `source_file_name`
- `created_count`
- `updated_count`
- `skipped_count`
- `error_count`
- `started_at`
- `finished_at`
- `error_message`

`import_job_errors`

- `id`
- `import_job_id`
- `sheet_name`
- `row_number`
- `field_name`
- `error_message`
- `raw_data`

### 审计

`audit_logs`

- `id`
- `family_id`
- `operator_id`
- `action`
- `resource_type`
- `resource_id`
- `success`
- `ip`
- `user_agent`
- `detail`
- `created_at`

## 认证与授权

认证流程：

1. 用户使用账号密码登录。
2. 后端先检查账号和访问 IP 是否处于临时封禁期，封禁中返回 429 和可读错误信息。
3. 后端校验密码哈希，失败时记录账号和访问 IP 维度失败次数。
4. 如果用户被禁用，拒绝登录。
5. 返回 `access_token` 和 `refresh_token`。
6. 后端保存 refresh token 哈希到 `user_sessions`。
7. 前端使用 access token 请求接口。
8. access token 过期后使用 refresh token 换取新 token。

密码找回策略：

- 不提供用户主动发起的忘记密码、自助找回密码接口或页面。
- 用户忘记密码时由管理员在线下核验后发起重置，重置后用户必须首次登录改密。
- 重置密码会废止用户现有 session，并清理该账号的登录失败记录。

授权规则：

- 所有业务接口必须从登录上下文获取 `user_id`、`family_id`、`role`。
- `super_admin` 可访问跨家庭管理接口。
- `family_admin` 和 `member` 只能访问本家庭数据。
- 后端必须做角色校验，前端隐藏菜单不能作为权限边界。

## API 设计原则

- 所有列表接口支持分页。
- 常用列表支持关键词搜索、状态筛选、日期范围筛选。
- 下拉接口返回系统预置项和当前家庭自定义项。
- 金额接口输入输出建议以元展示、以后端统一转换为分存储。
- 返回结构保持稳定，便于 Web 端和手机端复用。
- 涉及接口变更时同步维护 Swagger。

当前接口约定：

- 统一响应结构为 `{ code, msg, data }`，错误响应同样返回明确 `msg`。
- `POST /auth/login`、`POST /auth/refresh` 管理 Access Token 与 Refresh Token。
- `GET /version` 返回应用名称和版本号，用于前端展示、问题排查和发布验收。
- 应用版本号采用 `X.A.B.C` 四段式格式。每次提交代码前必须同步更新后端配置默认值、运行配置、前端兜底展示版本、Swagger 信息和 `CHANGELOG.md` 对应版本段。
- 家庭、用户、字典、往来对象、往来流水、审计日志、备份、导入恢复均提供 REST API。
- 列表接口返回 `{ total, list, page, page_size }`。
- 家庭级接口必须从登录上下文推导租户；`super_admin` 代家庭操作时通过 `family_id` 查询参数指定目标家庭。
- `GET /stats/overview` 支持 `start_date`、`end_date` 和 `family_id` 查询参数，并返回 `trend_granularity`。趋势数据在结果跨度不超过 90 天时按日聚合，否则按月聚合。

## 前端架构建议

技术栈：

- Vue 3
- Vite
- Naive UI
- ECharts
- lucide-vue-next

页面模块：

- 登录页
- 首次修改密码页
- 工作台
- 家庭管理
- 用户管理
- 往来对象管理
- 数据字典管理
- 礼尚往来记录
- 对象详情
- 统计分析
- 备份配置
- 导入恢复
- 审计日志

响应式策略：

- Web 端优先使用表格、筛选区、分页、弹窗表单。
- 手机端优先使用卡片列表、分组表单、底部导航和移动端图表布局。
- 手机端内容从顶部开始排列，底部导航固定在视口内并支持横向滚动。
- 手机端避免页面级横向溢出；必须展示表格时，仅表格容器内部横向滚动。
- 非当前页面有效的全局控件不展示，例如“搜索对象、手机号”仅在往来对象页展示。

视觉与状态体系：

- 前端样式以 `frontend/src/styles.css` 中的语义化 CSS token 为基础，统一背景、主色、边框、圆角、阴影和动效节奏。
- UI 风格采用现代轻玻璃后台视觉：背景可有低对比层次，业务容器保持高可读性，避免大面积高饱和渐变和过度装饰。
- KPI、备份摘要等指标区使用独立轻量信息卡，卡片通过淡边框、轻阴影、状态点和数字层级表达重点。
- Naive UI 组件允许通过 CSS 变量调整圆角和状态色，但不应把 `--n-color` 等颜色变量设置为渐变，避免按钮文字和状态渲染异常。
- 页面级异步加载使用骨架屏；按钮级异步操作使用 Naive UI 按钮 loading 状态。
- PC 端侧边栏使用固定满高背景层保证长页面中菜单区域铺满视口；移动端隐藏侧边栏菜单并使用底部固定导航。

项目协作与发布约束：

- 仓库内保留 `.codex/skills/project-release-discipline`，用于提交前检查文档、changelog、验证命令和中文提交信息。
- 每次形成可感知版本变化时，在 `CHANGELOG.md` 新增独立日期版本段，不把后续版本内容混入首版实现记录。
- 提交信息使用中文描述，允许保留 `feat`、`fix`、`docs` 等 conventional commit 类型前缀。

## 定时任务

第一版定时任务内置在 Go 后端进程中。

任务包括：

- 按家庭备份配置发送每日/每周/每月备份邮件。
- 每日清理 `user_sessions`。
- 可选清理临时导出文件。

定时任务实现要封装成 service，未来可迁移到独立 worker。

## 备份导入流程

### 定时备份

1. Scheduler 扫描启用的 `backup_configs`。
2. 判断是否到达执行时间。
3. 生成家庭级 Excel。
4. 如果启用加密，使用家庭 ZIP 密码生成加密 ZIP。
5. 发送邮件。
6. 写入 `backup_jobs`。

### 导入恢复

1. 上传 Excel。
2. 解析模板并执行 dry-run。
3. 返回新增、更新、跳过、错误、冲突统计。
4. 用户确认正式导入。
5. 系统先生成当前家庭备份，并发送邮件。
6. 备份邮件发送成功后执行合并导入。
7. 写入 `import_jobs`、`import_job_errors` 和 `audit_logs`。

## 安全约束

- 密码不得明文保存。
- 登录失败次数和临时封禁按账号、访问 IP 双维度控制，防止持续爆破。
- 忘记密码只能由管理员重置，不允许用户主动发起自助找回。
- Refresh Token 只保存哈希。
- ZIP 密码用 `APP_ENCRYPTION_KEY` 加密保存。
- 邮件正文不得包含 ZIP 密码。
- 所有家庭级数据查询必须带租户隔离条件。
- 备份、导入、用户管理、字典管理必须写审计日志。
