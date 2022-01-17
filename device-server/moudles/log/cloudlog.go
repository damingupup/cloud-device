/*
* @Author: 于智明
* @Date:   2021/2/3 3:50 下午
 */
package cloudlog

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"time"
)

var TimeStamp = time.Now().Format("2006-01-02")
var Logger *zap.Logger

func init() {
	Logger = getLog()
}

func getLog() *zap.Logger {
	Logger = ZapLogger()
	return Logger
}
func ZapLogger() *zap.Logger {
	var caller, development zap.Option

	// Option：基本日志选项
	//appName = zap.Fields(zap.String("app", "ios-proxy-video"))
	//version = zap.Fields(zap.String("version", "v0.1"))

	// Option：注释每条信息所在文件名和行号
	caller = zap.AddCaller()
	// Option：进入开发模式，使其具有良好的动态性能,记录死机而不是简单地记录错误。
	development = zap.Development()
	// 配置核心
	cores := getCore()
	return zap.New(cores, caller, development)
}

/**
获取zap core列表
*/
func getCore() (coreList zapcore.Core) {

	// 构建hook的 WriteSyncer 列表
	var infoWriteSyncerList []zapcore.WriteSyncer

	// 默认输出到文件和std
	rotateLogsTotalHook := getRotateLogsHook()
	infoWriteSyncerList = append(infoWriteSyncerList, zapcore.AddSync(os.Stdout), zapcore.AddSync(rotateLogsTotalHook))

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
获取rotateLogs Hook
*/
func getRotateLogsHook() io.Writer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./logs/" + TimeStamp + ".log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	return lumberJackLogger
}
