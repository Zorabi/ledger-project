package utils

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"stack", string(debug.Stack()),
				)

				c.JSON(http.StatusInternalServerError, ResponseData{
					Code: 500,
					Msg:  "服务器内部错误",
					Data: nil,
				})
				c.Abort()
			}
		}()
		if c != nil {
			c.Next()
		}
	}
}
