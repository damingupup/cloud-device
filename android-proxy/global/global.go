package global

const (
	OnLine        = "1"
	OffLine       = "0"
	Android       = 1
	Ios           = 2
	StatusSuccess = 1 //scrcpy启动成功
	StatusWait    = 2 //scrcpy启动等待
	StatusError   = 3 //scrcpy启动失败
	StatusRestart = 4 //scrcpy重启
	PingTime      = 15
	LogMsgLength  = 500
	ScrcpyPort    = "6612" // Scrcpy 启动端口
	ServerKey     = "sk1"  //为了和征文logcat服务名字尽量保持一致

)
