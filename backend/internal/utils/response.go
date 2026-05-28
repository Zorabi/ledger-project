package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type ResponseData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ResponseData{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func SuccessMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, ResponseData{
		Code: 0,
		Msg:  msg,
	})
}

func Fail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, ResponseData{
		Code: 1,
		Msg:  msg,
		Data: nil,
	})
}
