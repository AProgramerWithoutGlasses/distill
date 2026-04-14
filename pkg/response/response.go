package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// 通用响应结构体
type Response struct {
	Code Code   `json:"code"` // 状态码
	Msg  string `json:"msg"`  // 提示信息
	Data any    `json:"data"` // 数据
}

// 用于响应成功信息
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, &Response{
		Code: SuccessCode,
		Msg:  SuccessCode.Msg(), // code对应的提示信息
		Data: data,
	})
}

// 用于响应成功信息(单独的msg参数)
func SuccessWithMsg(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, &Response{
		Code: SuccessCode,
		Msg:  msg, // 手动编辑的提示信息
		Data: data,
	})
}

// 用于响应错误信息
func Fail(c *gin.Context, code Code) {
	c.JSON(http.StatusOK, &Response{
		Code: code,
		Msg:  code.Msg(), // 传入的code对应的提示信息
		Data: nil,
	})
}

// 用于响应错误信息（单独的msg参数）
func FailWithMsg(c *gin.Context, code Code, msg string) {
	c.JSON(http.StatusOK, &Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}
