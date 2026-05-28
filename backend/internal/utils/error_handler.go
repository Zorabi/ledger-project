package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 日志 (可以集成日志库)
				fmt.Println("PANIC:", err)

				// 返回统一的服务器内部错误响应
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
