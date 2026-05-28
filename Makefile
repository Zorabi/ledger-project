SHELL := /bin/sh

APP_NAME ?= ledger
BACKEND_DIR := backend
FRONTEND_DIR := frontend
COMPOSE ?= docker compose
DATABASE_URL ?= postgres://postgres:123456@127.0.0.1:5432/ledger?sslmode=disable
VERSION ?=
BACKUP_FILE ?= backups/$(APP_NAME)-$(shell date +%Y%m%d%H%M%S).dump

.PHONY: help test build frontend-build backend-build dev-backend dev-frontend docker-up docker-down docker-logs sql-apply db-backup db-restore

help:
	@echo "可用命令："
	@echo "  make test                 运行后端测试和前端构建"
	@echo "  make build                构建前后端 Docker 镜像"
	@echo "  make dev-backend          本地启动 Go 后端"
	@echo "  make dev-frontend         本地启动 Vite 前端"
	@echo "  make docker-up            使用 docker compose 启动全套服务"
	@echo "  make docker-down          停止 docker compose 服务"
	@echo "  make docker-logs          查看 docker compose 日志"
	@echo "  make sql-apply VERSION=x  执行 sql/<VERSION> 下的 DDL 和 data"
	@echo "  make db-backup            使用 pg_dump 备份 DATABASE_URL"
	@echo "  make db-restore           使用 pg_restore 恢复 BACKUP_FILE 到 DATABASE_URL"

test:
	cd $(BACKEND_DIR) && go test ./...
	cd $(FRONTEND_DIR) && npm run build

build: backend-build frontend-build

backend-build:
	docker build -t $(APP_NAME)-backend:local $(BACKEND_DIR)

frontend-build:
	cd $(FRONTEND_DIR) && npm ci && npm run build

dev-backend:
	cd $(BACKEND_DIR) && go run ./cmd/api

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev -- --host 0.0.0.0

docker-up:
	$(COMPOSE) up -d --build

docker-down:
	$(COMPOSE) down

docker-logs:
	$(COMPOSE) logs -f

sql-apply:
	@if [ -z "$(VERSION)" ]; then echo "请传入 VERSION，例如 make sql-apply VERSION=1.2.0.0"; exit 1; fi
	@if [ ! -d "sql/$(VERSION)" ]; then echo "sql/$(VERSION) 不存在"; exit 1; fi
	@for file in sql/$(VERSION)/*.sql; do \
		echo "Applying $$file"; \
		psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$$file"; \
	done

db-backup:
	@mkdir -p "$(dir $(BACKUP_FILE))"
	pg_dump --format=custom --file="$(BACKUP_FILE)" "$(DATABASE_URL)"
	@echo "Backup written to $(BACKUP_FILE)"

db-restore:
	@if [ ! -f "$(BACKUP_FILE)" ]; then echo "BACKUP_FILE 不存在：$(BACKUP_FILE)"; exit 1; fi
	pg_restore --clean --if-exists --no-owner --dbname="$(DATABASE_URL)" "$(BACKUP_FILE)"
