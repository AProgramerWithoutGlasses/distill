package server

import (
	"goweb_staging/pkg/response"
	"goweb_staging/service"

	"github.com/gin-gonic/gin"
)

func genContent(c *gin.Context) {
	var req service.GenContentReq

	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, response.ParamErrCode)
		return
	}

	content, err := svc.GenContent(req)
	if err != nil {
		response.FailWithMsg(c, response.ServerErrCode, err.Error())
		return
	}

	response.Success(c, content)
}
