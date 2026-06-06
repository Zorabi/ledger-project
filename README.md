# 礼尚往来

礼尚往来是一个家庭租户模式的礼金/礼品往来账本，包含 Go 后端、Vue 3 前端、PostgreSQL 数据库、Excel 备份导入和轻量审计能力。

当前应用版本：`1.4.0.0`

## 技术栈

- 后端：Go、Gin、GORM
- 前端：Vue 3、Vite、Naive UI、ECharts
- 数据库：PostgreSQL
- 部署：Docker / Docker Compose，或自行部署前后端进程

## 本地启动

前置依赖：

- Docker 和 Docker Compose
- 如需在当前机器直接操作数据库，需要安装 PostgreSQL client，确保本机可用 `psql`、`pg_dump`、`pg_restore`
- `Makefile` 主要用于本地开发、测试、运维机或 CI 封装，不是生产环境必需依赖

```bash
docker compose up -d --build
```

默认地址：

- 前端：http://localhost
- 后端：http://localhost:8080
- Swagger：http://localhost:8080/swagger/index.html
- 数据库：`127.0.0.1:5432/ledger`

也可以分开启动：

```bash
make dev-backend
make dev-frontend
```

## 关键环境变量

后端配置来自 `backend/config.yaml`，也支持通过环境变量覆盖。

| 环境变量 | 说明 |
| --- | --- |
| `DATABASE_POSTGRES_HOST` | PostgreSQL 地址 |
| `DATABASE_POSTGRES_PORT` | PostgreSQL 端口 |
| `DATABASE_POSTGRES_USER` | 数据库用户 |
| `DATABASE_POSTGRES_PASSWORD` | 数据库密码 |
| `DATABASE_POSTGRES_DBNAME` | 业务库名，建议使用 `ledger` |
| `DATABASE_POSTGRES_SSL_MODE` | SSL 模式，本地常用 `disable` |
| `DATABASE_POSTGRES_AUTO_CREATE` | 是否自动创建数据库，开发可开，生产共享实例建议关 |
| `DATABASE_POSTGRES_AUTO_MIGRATE` | 是否自动迁移表结构，开发可开，生产建议用 SQL 手动迁移 |
| `JWT_SECRET` | JWT 签名密钥，生产必须改强随机值 |
| `APP_ENCRYPTION_KEY` | 备份 ZIP 密码加密主密钥，迁移服务器时必须保留 |
| `APP_VERSION` | 应用展示版本号 |
| `LOG_LEVEL` | 日志级别：`debug`、`info`、`warn`、`error` |
| `LOG_CONSOLE` | 是否输出到标准输出 |
| `LOG_FILE` | 是否写入日志文件 |
| `LOG_FILE_PATH` | 日志文件路径，默认 `logs/ledger-backend.log` |
| `LOG_MAX_SIZE_MB` | 单个日志文件最大容量（MB） |
| `LOG_MAX_BACKUPS` | 最多保留的轮转文件数量 |
| `LOG_MAX_AGE_DAYS` | 日志文件最长保留天数 |
| `LOG_COMPRESS` | 是否压缩已轮转的日志文件 |

## 后端日志

后端默认同时输出 JSON 日志到控制台和 `logs/ledger-backend.log`。日志包含应用事件、HTTP 请求、状态码、耗时、客户端 IP、User-Agent 和 `trace_id`。

```yaml
log:
  level: "info"
  console: true
  file: true
  file_path: "logs/ledger-backend.log"
  max_size_mb: 100
  max_backups: 10
  max_age_days: 30
  compress: true
```

相对 `file_path` 以实际读取到的 `config.yaml` 所在目录为基准，因此本地默认生成在 `backend/logs/ledger-backend.log`，Docker 中生成在 `/app/logs/ledger-backend.log`。Docker Compose 已将 `/app/logs` 挂载为 `backend-logs` 数据卷，容器重建后日志仍会保留。若生产环境统一采集标准输出，可设置 `LOG_FILE=false`；若只保留文件日志，可设置 `LOG_CONSOLE=false`。

## 部署建议

### 独立 Docker Compose 部署

适合单机或内网小规模部署：

```bash
docker compose up -d --build
docker compose logs -f
```

生产环境必须修改：

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `APP_ENCRYPTION_KEY`
- email 邮件配置

### 共享 PostgreSQL 实例部署

推荐“共享 PostgreSQL 实例，不共享业务 database”。也就是说，同一台 PostgreSQL 可以给多个项目使用，但本系统应拥有独立数据库，例如 `ledger`，避免表名、权限和迁移互相影响。

由 DBA 或管理员先创建数据库和用户：

```sql
CREATE USER ledger_app WITH PASSWORD 'change_me';
CREATE DATABASE ledger OWNER ledger_app;
GRANT CONNECT ON DATABASE ledger TO ledger_app;
```

然后在能访问生产数据库的环境中按版本执行 SQL。首次部署需要从 `sql/1.0.0.0` 开始，按版本从小到大执行到目标版本；每个版本目录内先执行 `001_ddl.sql`，再执行 `002_data.sql`。生产环境不要求安装或使用本仓库的 `Makefile`，DBA 或运维可以直接使用 `psql` 执行：

```bash
psql 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/<version>/001_ddl.sql
psql 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/<version>/002_data.sql
```

应用环境变量建议：

```bash
DATABASE_POSTGRES_AUTO_CREATE=false
DATABASE_POSTGRES_AUTO_MIGRATE=false
DATABASE_POSTGRES_DBNAME=ledger
```

如果必须和其他项目共用同一个 database，而不是共用 PostgreSQL 实例，目前不建议直接部署。本项目当前没有表名前缀或独立 schema 配置，存在表名冲突和权限边界不清的问题。

### 前端静态资源压缩

前端生产构建会在 `frontend/dist/assets/` 中生成原始静态资源以及 `.gz`、`.br` 预压缩文件。Nginx 只会按浏览器能力返回其中一种文件，不会让用户同时下载多份。

如果 Nginx 未编译 Brotli 模块，只开启 gzip 静态文件即可：

```nginx
location / {
    root /path/to/frontend/dist;
    try_files $uri $uri/ /index.html;
    gzip_static on;
}
```

如果 Nginx 已编译 Brotli 模块，可同时开启：

```nginx
location / {
    root /path/to/frontend/dist;
    try_files $uri $uri/ /index.html;
    gzip_static on;
    brotli_static on;
}
```

未编译 Brotli 模块时不要写 `brotli_static on;`，否则 `nginx -t` 会因未知指令失败。

### SQL 执行环境

首次部署、版本升级和服务器迁移都可能需要执行 SQL、备份或恢复。生产数据库无法从开发机直连时，不需要把生产库暴露出来，选择一种符合公司运维规范的方式即可：

- 将 `sql/<version>/` 目录交给 DBA 或运维，由他们在数据库管理环境中按 `001_ddl.sql`、`002_data.sql` 顺序执行。
- 在应用服务器、堡垒机或 CI/CD 内网任务中安装 PostgreSQL client 后执行 `psql`。
- 通过 SSH 隧道临时映射数据库端口后，在本机执行 `psql`：

  ```bash
  ssh -L 15432:db.internal:5432 ops@bastion
  psql 'postgres://ledger_app:change_me@127.0.0.1:15432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.2.0.0/001_ddl.sql
  psql 'postgres://ledger_app:change_me@127.0.0.1:15432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.2.0.0/002_data.sql
  ```

- 在与数据库同 Docker 网络的临时 PostgreSQL client 容器中执行 `psql`。

后续“数据库升级”和“服务器迁移”中的命令，都默认在上述任一可访问数据库的环境中执行。

## 数据库升级

### 开发环境

可以保持：

```bash
DATABASE_POSTGRES_AUTO_CREATE=true
DATABASE_POSTGRES_AUTO_MIGRATE=true
```

服务启动时会自动创建数据库并通过 GORM `AutoMigrate` 补齐新增表和字段。

### 生产环境

建议使用手动 SQL 模式：

1. 停止写入或进入维护窗口。
2. 执行数据库备份。
3. 阅读新版本 `sql/<version>/001_ddl.sql` 和 `sql/<version>/002_data.sql`。
4. 按版本顺序执行 SQL。
5. 部署新版本应用。
6. 验证登录、列表、录入、备份导出和审计日志。

备份示例：

```bash
pg_dump --format=custom --file=backups/ledger-20260528.dump 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require'
```

升级到某个新版本时，只执行当前运行版本之后的新版本 SQL。例如从 `1.1.0.1` 升级到 `1.2.0.0`：

```bash
psql 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.2.0.0/001_ddl.sql
psql 'postgres://ledger_app:change_me@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.2.0.0/002_data.sql
```

新增字段时应优先使用兼容迁移：

- 先新增可空字段或带默认值字段。
- 回填历史数据。
- 再按需要增加 `NOT NULL` 或唯一约束。
- 删除字段、改类型、重命名字段属于高风险变更，必须提供回滚方案。

## 服务器迁移

迁移到新服务器时，重点是数据库和密钥。

1. 在旧服务器停止写入或进入维护窗口。
2. 在能访问旧数据库的环境备份数据库：

   ```bash
   pg_dump --format=custom --file=backups/ledger.dump 'postgres://ledger_app:change_me@old-host:5432/ledger?sslmode=require'
   ```

3. 复制代码、配置、备份文件到新服务器。
4. 在新服务器创建数据库和用户。
5. 在能访问新数据库的环境恢复数据库：

   ```bash
   pg_restore --clean --if-exists --no-owner --dbname='postgres://ledger_app:change_me@new-host:5432/ledger?sslmode=require' backups/ledger.dump
   ```

6. 确认新服务器使用同一个 `APP_ENCRYPTION_KEY`。如果该密钥丢失，历史备份 ZIP 密码无法解密。
7. 配置 `JWT_SECRET`。如果变更 JWT 密钥，历史登录会话会失效，用户需要重新登录。
8. 启动服务并验证核心流程。
9. 切换域名、反向代理或负载均衡。

## SQL 目录规范

SQL 位于 `sql/<version>/`：

- `001_ddl.sql`：表结构、索引、字段。
- `002_data.sql`：系统预置数据或版本标记。

每次涉及表结构、索引、字段或预置数据的提交，都必须随版本新增对应 SQL 文件，并同步更新 `CHANGELOG.md`。

## 本地和运维辅助命令

以下命令用于开发机、测试环境、运维机或 CI。生产环境可以不用 `make`，直接按上文使用 `psql`、`pg_dump`、`pg_restore`。

```bash
make help
make test
make build
make build-linux-amd64
make build-linux-arm64
make build-darwin-arm64
make build-windows-amd64
make build-windows-arm64
make build-all
make docker-build
make docker-up
make docker-down
make docker-logs
make sql-apply VERSION=1.2.0.0
make db-backup
make db-restore BACKUP_FILE=backups/ledger.dump
```

构建产物说明：

- 所有发布产物统一输出到根目录 `dist/`。
- 后端二进制按平台输出到 `dist/backend/<系统>-<架构>/`，文件名统一为 `ledger-backend`，Windows 为 `ledger-backend.exe`。例如 `dist/backend/linux-amd64/ledger-backend`、`dist/backend/windows-amd64/ledger-backend.exe`。
- 前端静态资源先构建到 `frontend/dist/`，再打包为 `dist/frontend/ledger-web.zip`。
- `make build`：构建当前平台后端二进制，并生成前端 zip 包。
- `make build-linux-amd64` / `make build-linux-arm64`：构建 Linux 后端二进制。
- `make build-darwin-amd64` / `make build-darwin-arm64`：构建 macOS 后端二进制。
- `make build-windows-amd64` / `make build-windows-arm64`：构建 Windows 后端二进制。
- `make build-all`：构建 Linux amd64/arm64、macOS amd64/arm64、Windows amd64/arm64 后端二进制，并生成前端 zip 包。
- `make backend-build GOOS=linux GOARCH=amd64`：高级自定义入口，按任意 Go 支持的平台构建。
- `make docker-build`：分别构建前后端 Docker 镜像。
- `make clean`：清理根目录 `dist/` 产物。
