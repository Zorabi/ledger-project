# 使用多阶段构建来减小镜像体积
# 1. 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
# 下载依赖
RUN go mod download

COPY . .

# 编译应用，-o 指定输出文件名，-ldflags "-w -s" 去除调试信息
RUN CGO_ENABLED=0 GOOS=linux go build -o main -ldflags "-w -s" ./cmd/api

# 2. 运行阶段
FROM alpine:latest

WORKDIR /app

# 从构建阶段拷贝编译好的二进制文件和配置文件
COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["/app/main"]
