package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.New().String()
		requestLogger := slog.With("trace_id", traceID)
		c.Set("logger", requestLogger)
		c.Header("X-Trace-ID", traceID)

		startedAt := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}
		requestLogger.LogAttrs(c.Request.Context(), level, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("query", c.Request.URL.RawQuery),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(startedAt).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("errors", c.Errors.String()),
		)
	}
}
