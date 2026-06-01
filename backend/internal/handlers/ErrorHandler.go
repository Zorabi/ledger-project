package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	return e.Message
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		err := c.Errors.Last()
		if err != nil {
			return
		}

		var appError *AppError
		if errors.As(err.Err, &appError) {
			c.JSON(http.StatusOK, gin.H{
				"code": appError.Code,
				"msg":  appError.Message,
			})
		} else {
			// 对于未知的 panic 或其他错误
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "服务器内部错误",
			})
		}
	}
}
