package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"ios-proxy/common"
	"time"
)

func LoggerToFile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		ctx.Next()

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := ctx.Request.Method

		// 请求路由
		reqUri := ctx.Request.RequestURI

		// 状态码
		statusCode := ctx.Writer.Status()

		// 请求IP
		clientIP := ctx.ClientIP()
		common.Log.Info("", zap.Int("statusCode", statusCode),
			zap.String("latencyTime", latencyTime.String()),
			zap.String("clientIP", clientIP),
			zap.String("reqMethod", reqMethod),
			zap.String("reqUri", reqUri))
	}
}
