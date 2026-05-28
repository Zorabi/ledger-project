package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
)

func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 为每个请求生成一个唯一的 Trace ID
		traceID := uuid.New().String()

		// 创建一个带特定字段的 logger 实例
		requestLogger := slog.With("trace_id", traceID)

		// 使用 c.Set 将 logger 实例存入上下文，方便后续 Handler 使用
		c.Set("logger", requestLogger)

		c.Next()
	}
}