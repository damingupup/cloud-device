package common

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
)

var Log *zap.Logger

func GetLog(uid string) *zap.Logger {
	Log = ZapLogger(uid)
	return Log
}

/*
项目中使用的通用zap日志记录器，返回logger
*/
func ZapLogger(uid string) *zap.Logger {
	var caller, development zap.Option

	// Option：基本日志选项
	//appName = zap.Fields(zap.String("app", "ios-proxy"))
	//version = zap.Fields(zap.String("version", "v0.1"))

	// Option：注释每条信息所在文件名和行号
	caller = zap.AddCaller()
	// Option：进入开发模式，使其具有良好的动态性能,记录死机而不是简单地记录错误。
	development = zap.Development()
	// 配置核心
	cores := getCore(uid)
	return zap.New(cores, caller, development)
}

/**
获取zap core列表
*/
func getCore(uid string) (coreList zapcore.Core) {

	// 构建hook的 WriteSyncer 列表
	var infoWriteSyncerList []zapcore.WriteSyncer

	// 默认输出到文件和std
	rotatelogsTotalHook := getRotatelogsHook(uid)
	infoWriteSyncerList = append(infoWriteSyncerList, zapcore.AddSync(os.Stdout), zapcore.AddSync(rotatelogsTotalHook))

	return zapcore.NewCore(
		// 编码器配置
		getEncoder(),
		// 打印到控制台和文件
		zapcore.NewMultiWriteSyncer(infoWriteSyncerList...),
		// 日志级别
		zapcore.DebugLevel)
}
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.CallerKey = "extra"
	encoderConfig.StacktraceKey = "error"
	return zapcore.NewJSONEncoder(encoderConfig)
}

/*
获取rotatelogs Hook
*/
func getRotatelogsHook(uid string) io.Writer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./logs/" + uid + "/" + TimeStamp + ".log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	return lumberJackLogger
}
