package server

import (
	"goweb_staging/logger"
	"goweb_staging/middleware"
	"goweb_staging/pkg/settings"
	"goweb_staging/service"

	"github.com/gin-gonic/gin"
)

var svc *service.Service

func initRouter() *gin.Engine {
	// 创建一个默认的路由引擎
	r := gin.Default()

	// 注册使用的中间件
	r.Use(logger.GinLogger(), logger.GinRecovery(true), middleware.CORSMiddleware())

	// jwt认证中间件
	r.Use(middleware.JWTAuthMiddleware())

	r.POST("/test", test)

	return r
}

func Init(app *settings.AppConfig) *gin.Engine {
	svc = service.InitService(app)
	return initRouter()
}
