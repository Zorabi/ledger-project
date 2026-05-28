# SQL 版本目录

本目录用于生产环境手动建库、建表和升级字段。目录名与应用版本号一致，每个版本目录按顺序提供：

- `001_ddl.sql`：表结构、字段、索引等 DDL。
- `002_data.sql`：系统预置数据、版本标记等 DML。

执行原则：

1. 新部署按版本从小到大执行所有 SQL。
2. 升级部署只执行当前运行版本之后的新版本 SQL。
3. 生产共享 PostgreSQL 实例建议设置：
   - `DATABASE_POSTGRES_AUTO_CREATE=false`
   - `DATABASE_POSTGRES_AUTO_MIGRATE=false`
4. 执行 SQL 前必须先做数据库备份。
5. 生产数据库通常不允许开发机直连，SQL 应在应用服务器、堡垒机、CI/CD 内网任务、DBA 工作台，或通过 SSH 隧道后的本机执行。
6. 生产环境不依赖 `Makefile`，可以直接使用 `psql` 执行 SQL 文件。

示例：

```bash
psql 'postgres://ledger_app:password@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.0.0.0/001_ddl.sql
psql 'postgres://ledger_app:password@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.0.0.0/002_data.sql
psql 'postgres://ledger_app:password@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.1.0.0/001_ddl.sql
psql 'postgres://ledger_app:password@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.1.0.0/002_data.sql
psql 'postgres://ledger_app:password@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.2.0.0/001_ddl.sql
psql 'postgres://ledger_app:password@db-host:5432/ledger?sslmode=require' -v ON_ERROR_STOP=1 -f sql/1.2.0.0/002_data.sql
```
