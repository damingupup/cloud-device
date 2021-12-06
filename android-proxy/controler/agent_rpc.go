package controler

import (
	"context"
	"crypto/rand"
	"ctp-android-proxy/configs"
	"ctp-android-proxy/global"
	"ctp-android-proxy/moudles/adbclient"
	cloudLog "ctp-android-proxy/moudles/log"
	model "ctp-android-proxy/moudles/modle"
	"ctp-android-proxy/moudles/shell"
	"ctp-android-proxy/moudles/tools"
	pb "ctp-android-proxy/proto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/DeanThompson/syncmap"
	"github.com/hpcloud/tail"
	"go.uber.org/zap"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var logSingle *logInfo

func init() {
	fmt.Println("初始化logSingle")
	logSingle = new(logInfo)
}

func TempFileName(dir, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(dir, hex.EncodeToString(randBytes)+suffix)
}

var background = &tools.Background{
	Sm: syncmap.New(),
}

type logMsg struct {
	Pid      string `json:"pid"`
	Name     string `json:"name"`
	Level    string `json:"level"`
	DeviceId string `json:"deviceid"`
	Time     string `json:"time"`
	Message  string `json:"message"`
	Msg      string `json:"msg"`
	HoldId   string `json:"holdid"`
	LogcatId string `json:"logcatid"`
}
type logInfo struct {
	status bool  //log开关状态
	logId  int32 //云测的日志id
	name   string
	stop   chan bool
	Cmd    *shell.Service
	Id     int32 //数据库的id
}

type DeviceAgent struct {
}

//检查启动情况
func (d *DeviceAgent) Ping(context.Context, *pb.PingRequest) (*pb.PingResponse, error) {
	var status pb.PingResponseStatus
	switch tools.AndroidControl.Status {
	case global.StatusSuccess:
		cloudLog.Logger.Info("start success")
		status = pb.PingResponse_Success
	case global.StatusWait:
		cloudLog.Logger.Info("start wait")
		status = pb.PingResponse_Wait
	case global.StatusError:
		cloudLog.Logger.Error("start error")
		status = pb.PingResponse_Fail
	}
	return &pb.PingResponse{Status: status,
		VideoPort:   tools.AndroidControl.AgentPort,
		ControlPort: tools.AndroidControl.AgentPort}, nil
}

//视频流
func (d *DeviceAgent) VideoStream(_ *pb.VideoStreamRequest, stream pb.DeviceAgentService_VideoStreamServer) error {
	video := tools.AndroidControl.Video.Subscribe()
	for {
		select {
		case picData, _ := <-video:
			err := stream.Send(&pb.VideoStreamResponse{PicBytes: picData})
			if err != nil {
				cloudLog.Logger.Error("数据大小", zap.Int("length", len(picData)))
				cloudLog.Logger.Error("数据传输失败", zap.String("err", err.Error()))
				return nil
			}
		}
	}
}

//控制
func (d *DeviceAgent) ControlStream(stream pb.DeviceAgentService_ControlStreamServer) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			return err
		}
		req := tools.Message{}
		err = json.Unmarshal([]byte(resp.Command), &req)
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			err = stream.Send(&pb.ControlStreamResponse{Result: "指令错误"})
			if err != nil {
				cloudLog.Logger.Error(err.Error())
				return err
			}
			continue
		}
		tools.AndroidControl.ControlStream <- req
	}
}

//安装app
func (d *DeviceAgent) InstallApp(r *pb.InstallAppRequest, stream pb.DeviceAgentService_InstallAppServer) error {
	tmpdir := "tmp/" + tools.AndroidControl.Uid
	filePath := TempFileName(tmpdir, ".apk")
	url := r.Url
	key := background.HTTPDownload(url, filePath, 0644)
	state := background.Get(key)
	go func() {
		defer os.Remove(filePath) // release sdcard space
		state.Status = "downloading"
		if err := background.Wait(key); err != nil {
			state.Error = err.Error()
			state.Message = "http download error"
			state.Status = "failure"
			return
		}
		state.Message = "installing"
		state.Status = "installing"
		err := forceInstallAPK(filePath, tools.AndroidControl.Uid)
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			state.Error = err.Error()
			state.Message = "error install"
			state.Status = "failure"
		} else {
			state.Message = "success installed"
			state.Status = "success"
		}
	}()
	//推送安装进度
	for {
		time.Sleep(time.Second)
		data, _ := json.Marshal(state)
		err := stream.Send(&pb.InstallAppStreamResponse{Result: string(data)})
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			break
		}
		if !(state.Status == "downloading" || state.Status == "installing") {
			break
		}
	}
	return nil
}

func (d *DeviceAgent) LogStream(stream pb.DeviceAgentService_LogStreamServer) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			cloudLog.Logger.Error(err.Error())
			return err
		}
		if resp.Status {
			if logSingle.logId == 0 {
				logSingle.logId = resp.LogId
			}
			if logSingle.status {
				//log已经启动
				cloudLog.Logger.Warn("日志已经启动")
				err = stream.Send(&pb.LogStreamResponse{Data: "已经启动"})
				if err != nil {
					cloudLog.Logger.Info(err.Error())
				}
				continue
			}
			logSingle.status = true
			go d.startLog(stream, resp)
		} else {
			go d.stopLog(stream)
		}
	}
}

func (d *DeviceAgent) startLog(stream pb.DeviceAgentService_LogStreamServer, r *pb.LogStreamRequest) {
	logSingle.Id = 0 //新开log将主键置为0
	logSingle.stop = make(chan bool)
	snowId := tools.SnowId()
	uid := tools.AndroidControl.Uid
	logIdStr := strconv.FormatInt(int64(r.LogId), 10)
	logPath := path.Join(`./`, `data`, uid, logIdStr)
	logKey := global.ServerKey + snowId
	logSingle.name = path.Join(logPath, logKey+".log")
	if !tools.PathExists(logPath) {
		os.MkdirAll(logPath, os.ModePerm)
	}
	mysql := global.MysqlDb
	newLog := model.FileModel{UserId: int(r.UserId), GroupId: int(r.GroupId),
		LogId: int(r.LogId), Name: logKey + ".log"}
	result := mysql.Create(&newLog)
	if result.Error != nil || result.RowsAffected != 1 {
		cloudLog.Logger.Error(result.Error.Error(), zap.String("log", "日志记录失败"))
		logSingle.status = false
		err := stream.Send(&pb.LogStreamResponse{Data: "日志记录创建失败"})
		if err != nil {
			cloudLog.Logger.Error(err.Error())
		}
		return
	}
	logSingle.Id = newLog.ID
	wg := sync.WaitGroup{}
	wg.Add(1)
	go d.execLogShell(&wg)
	wg.Wait()
	var logfile *tail.Tail
	defer func() {
		cloudLog.Logger.Info("关闭日志文件")
		if logSingle != nil {
			logfile.Stop()
		}

	}()
	var err error
	name := strings.Replace(logSingle.name, ".log", ".json", 1)
	logfile, err = tail.TailFile(name, tail.Config{Follow: true})
	if err != nil {
		err = stream.Send(&pb.LogStreamResponse{Data: "日志读取失败"})
		if err != nil {
			cloudLog.Logger.Error(err.Error())
		}
		logSingle.status = false
		return
	}
	content := [global.LogMsgLength]*logMsg{}
	logTimeStart := time.Now().Unix()
	logIndex := 0
	defer func() {
		cloudLog.Logger.Info("尝试一下日志上传")
		//上传日志并回调日志服务，清理本地文件
		d.uploadLog(name, snowId)
	}()
loop:
	for {
		if logIndex == global.LogMsgLength || time.Now().Unix()-logTimeStart > 1 {
			err = d.sendLog(content, stream)
			if err != nil {
				cloudLog.Logger.Error(err.Error())
				d.stopLog(stream)
				break loop
			}
			content = [global.LogMsgLength]*logMsg{}
			logTimeStart = time.Now().Unix()
			logIndex = 0
		}
		select {
		case _, ok := <-logSingle.stop:
			if !ok {
				logSingle.status = false
				cloudLog.Logger.Info("释放日志")
				break loop
			}
		case line, _ := <-logfile.Lines:
			logTmp := logMsg{}
			err = json.Unmarshal([]byte(line.Text), &logTmp)
			if err != nil {
				logTmp.Message = line.Text
			}
			content[logIndex] = &logTmp
			logIndex += 1
		}
	}

}

func (d *DeviceAgent) sendLog(msg [global.LogMsgLength]*logMsg, stream pb.DeviceAgentService_LogStreamServer) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		msgBytes = []byte{}
	}
	msgStr := string(msgBytes)
	err = stream.Send(&pb.LogStreamResponse{Data: msgStr})
	if err != nil {
		cloudLog.Logger.Error(err.Error())
		return err
	}
	return nil
}
func (d *DeviceAgent) execLogShell(wg *sync.WaitGroup) {
	cmdStr := fmt.Sprintf(`%s -s %s logcat >> %s`, configs.CloudConfig.Server.AdbPath, tools.AndroidControl.Uid, logSingle.name)
	logCmd := shell.Service{
		Log:  cloudLog.Logger,
		Name: "sh",
		Args: []string{"-c", cmdStr},
	}
	logSingle.status = true
	logSingle.Cmd = &logCmd
	wg.Done()
	logCmd.Run()

}
func (d *DeviceAgent) stopLog(stream pb.DeviceAgentService_LogStreamServer) {
	//关闭log，首先将log的进程杀掉
	cloudLog.Logger.Info("我要关闭log了")
	if !logSingle.status {
		cloudLog.Logger.Info("日志已经关闭")
		err := stream.Send(&pb.LogStreamResponse{Data: "日志读取失败"})
		if err != nil {
			cloudLog.Logger.Error(err.Error())
		}
		return
	}
	logSingle.Cmd.Stop()
	logSingle.status = false
	close(logSingle.stop)
}

func (d *DeviceAgent) uploadLog(josnPath string, snowId string) {
	////上传日志文件
	logPath := strings.Replace(josnPath, ".json", ".log", 1)
	file := tools.FileResult{}
	err := tools.UploadFile(logPath, &file)
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
	db := global.MysqlDb
	size := strconv.FormatInt(file.Size, 10)
	result := db.Model(&model.FileModel{}).Where("id = ?", logSingle.Id).Updates(model.FileModel{Domain: file.Domain, Path: file.Path, Md5Id: file.Md5, Size: size, Type: "log"})
	cloudLog.Logger.Info("我要修改日志记录了啊")
	if result.RowsAffected != 1 {
		cloudLog.Logger.Error("日志添加失败" + tools.AndroidControl.Uid)
		return
	}
	err = os.RemoveAll(path.Dir(josnPath))
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
}

func forceInstallAPK(filepath string, uid string) error {
	am := &tools.APKManager{Path: filepath}
	am.Uid = uid
	return am.ForceInstall()
}

func (d *DeviceAgent) RemoteDebug(ctx context.Context, r *pb.RemoteDebugRequest) (*pb.RemoteDebugResponse, error) {
	//远程调试
	var port string
	switch r.Status {
	case true:
		if tools.AndroidControl.RemoteDebug == nil {
			port = strconv.Itoa(tools.GetFreePort())
			args := []string{configs.CloudConfig.Server.AdbKitPath, "usb-device-to-tcp", "-p", port, tools.AndroidControl.Uid}
			wg := sync.WaitGroup{}
			wg.Add(1)
			cmd := shell.Service{Name: "node", Args: args, Wg: &wg}
			go cmd.Run()
			wg.Wait()
			tools.AndroidControl.RemoteDebug = &cmd
		} else {
			port = tools.AndroidControl.RemotePort
		}
	case false:
		if tools.AndroidControl.RemoteDebug != nil {
			tools.AndroidControl.RemoteDebug.Stop()
		}
		port = ""
		tools.AndroidControl.RemoteDebug = nil
	}

	return &pb.RemoteDebugResponse{Port: port, Host: configs.CloudConfig.Server.Host}, nil
}

func (d *DeviceAgent) ResetEnv(ctx context.Context, r *pb.ResetEnvRequest) (*pb.ResetEnvResponse, error) {
	//重置app环境
	//恢复设置选项
	//adb := shell.Adb{Uid: tools.AndroidControl.Uid, Wait: true}
	//adb.Shell([]string{"pm", "enable", "com.android.settings"})
	db := global.MysqlDb
	device := model.DeviceModel{SerialId: tools.AndroidControl.Uid}
	db.First(&device)
	//过快断开会偶现一些奇怪的问题，所以在这停顿3秒
	time.Sleep(time.Second * 3)
	//if device.Apk != "" {
	//	adb.Shell([]string{"pm", "list", "packages", "-3"})
	//	result := adb.Result()
	//	result = strings.Replace(result, "package:", "", -1)
	//	result = strings.TrimSpace(result)
	//	apps := strings.Split(result, "\n")
	//	for _, v := range apps {
	//		if !strings.Contains(device.Apk, v) {
	//			adb.Shell([]string{"pm", "uninstall", v})
	//		}
	//	}
	//}
	return &pb.ResetEnvResponse{Status: true}, nil
}

func (d *DeviceAgent) VerifyCode(ctx context.Context, r *pb.VerifyCodeRequest) (*pb.VerifyCodeResponse, error) {
	var status = true
	if r.Code != tools.AndroidControl.Code {
		status = false
		cloudLog.Logger.Warn("非法访问")
	}
	return &pb.VerifyCodeResponse{Status: status}, nil
}

func (d *DeviceAgent) Stop(ctx context.Context, r *pb.StopRequest) (*pb.StopResponse, error) {
	//if r.Code != tools.AndroidControl.Code {
	//	cloudLog.Logger.Warn("非法访问")
	//} else {
	//
	//}、
	adb := shell.Adb{Uid: tools.AndroidControl.Uid, Wait: true}
	adb.Shell([]string{"rm", "-rf", "/data/local/tmp/*"})
	adb.RemoveForward(tools.AndroidControl.AgentPort)
	tools.AndroidControl.Server.Stop()
	return &pb.StopResponse{}, nil

}

func (d *DeviceAgent) ShellStream(stream pb.DeviceAgentService_ShellStreamServer) error {
	ctb, cancel := context.WithCancel(context.Background())
	adb := adbclient.AdbClient{Uid: tools.AndroidControl.Uid}
	defer func() {
		cloudLog.Logger.Info("都退出了啊")
		cloudLog.Logger.Info("》》》》》》》》》》》》》》》》》》》")
		cancel()
		adb.Stop()
	}()
	keep := false
	ctx, cancelChild := context.WithCancel(ctb)
	defer cancelChild()
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		switch msg.Command {
		case "exit":
			adb.Stop()
			keep = false
		default:
			if keep {
				continue
			}
			keep = true
			adb.Result = make(chan string, 1024*1024*2)
			go adb.Shell(ctx, msg.Command)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case data, ok := <-adb.Result:
						if ok != true {
							keep = false
							return
						}
						err = stream.Send(&pb.ShellStreamResponse{Result: data})
						if err != nil {
							cancel()
							return
						}
					}
				}
			}()
		}

	}
}

func (d *DeviceAgent) PushFile(ctx context.Context, r *pb.PushFileRequest) (*pb.PushFileResponse, error) {
	tmpdir := "tmp/" + tools.AndroidControl.Uid
	filePath := TempFileName(tmpdir, ".tmp")
	url := r.Url
	key := background.HTTPDownload(url, filePath, 0644)
	state := background.Get(key)
	defer os.Remove(filePath)
	state.Status = "downloading"
	if err := background.Wait(key); err != nil {
		state.Error = err.Error()
		state.Message = "http download error"
		state.Status = "failure"
		return &pb.PushFileResponse{Result: err.Error()}, err
	}
	adb := shell.Adb{Uid: r.Uid, Wait: true}
	adb.Shell([]string{"mkdir", "/sdcard/cloud"})
	savePath := path.Join("/", "sdcard", "cloud", r.Name)
	cloudLog.Logger.Info(">>>>>>>>>>>>>>>>>>>>>>")
	cloudLog.Logger.Info(savePath)
	cloudLog.Logger.Info(">>>>>>>>>>>>>>>>>>>>>>")
	adb.Push(filePath, savePath)
	return &pb.PushFileResponse{}, nil
}
