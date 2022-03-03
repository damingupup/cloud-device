/*
* @Author: 于智明
* @Date:   2021/2/20 11:16 上午
 */
package controler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hpcloud/tail"
	"go.uber.org/zap"
	"io/ioutil"
	"ios-proxy/common"
	"ios-proxy/configs"
	"ios-proxy/moudles/model"
	pb "ios-proxy/proto"
	"ios-proxy/utils"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var logSingle *logInfo

func init() {
	fmt.Println("初始化logSingle")
	logSingle = new(logInfo)
}

type logMsg struct {
	Pid      string `json:"pid"`
	Name     string `json:"name"`
	Level    string `json:"level"`
	Deviceid string `json:"deviceid"`
	Time     string `json:"time"`
	Message  string `json:"message"`
	Msg      string `json:"msg"`
	Holdid   string `json:"holdid"`
	Logcatid string `json:"logcatid"`
}
type logInfo struct {
	status bool  //log开关状态
	logId  int32 //云测的日志id
	name   string
	stop   chan bool
	cmd    common.CmdService
	Id     int32 //数据库的id
}

type DeviceAgent struct {
}

//检查启动情况
func (d *DeviceAgent) Ping(context.Context, *pb.PingRequest) (*pb.PingResponse, error) {
	fmt.Println(utils.WDAControl.Status)
	var status pb.PingResponseStatus
	switch utils.WDAControl.Status {
	case configs.StatusSuccess:
		common.Log.Info("start success")
		status = pb.PingResponse_Success
	case configs.StatusWait:
		common.Log.Info("start wait")
		status = pb.PingResponse_Wait
	case configs.StatusError:
		common.Log.Error("start error")
		status = pb.PingResponse_Fail
	}
	return &pb.PingResponse{Status: status,
		VideoPort:   utils.WDAControl.AgentPort,
		ControlPort: utils.WDAControl.ControlPort}, nil
}

//视频流
func (d *DeviceAgent) VideoStream(_ *pb.VideoStreamRequest, stream pb.DeviceAgentService_VideoStreamServer) error {
	video := utils.WDAControl.Video.Subscribe()
	for {
		select {
		case picData, _ := <-video:
			err := stream.Send(&pb.VideoStreamResponse{PicBytes: picData})
			if err != nil {
				common.Log.Error("数据传输失败", zap.String("err", err.Error()))
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
			common.Log.Error(err.Error())
			return err
		}
		go d.handCommand(stream, resp.Command)
	}
}
func (d *DeviceAgent) handCommand(stream pb.DeviceAgentService_ControlStreamServer, command string) {
	host := "http://127.0.0.1:" + utils.WDAControl.ControlPort
	moveUrl := strings.Join([]string{host, "actions"}, "/")
	homeUrl := strings.Join([]string{host, "wda", "pressButton"}, "/")
	sizeUrl := strings.Join([]string{host, "window", "size"}, "/")
	var url string
	if strings.Contains(command, "home") {
		url = homeUrl
	} else if strings.Contains(command, "size") {
		url = sizeUrl
	} else {
		url = moveUrl
	}
	body := bytes.NewReader([]byte(command))
	var resp *http.Response
	var err error
	if strings.Contains(url, "size") {
		resp, err = http.Get(url)
		if err != nil {
			common.Log.Error(err.Error())
			err = stream.Send(&pb.ControlStreamResponse{Result: "执行失败"})
			if err != nil {
				common.Log.Error(err.Error())
			}
			return
		}
	} else {
		resp, err = http.Post(url, "application/json", body)
		if err != nil {
			common.Log.Error(err.Error())
			err = stream.Send(&pb.ControlStreamResponse{Result: "执行失败"})
			if err != nil {
				common.Log.Error(err.Error())
			}
			return
		}
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	err = stream.Send(&pb.ControlStreamResponse{Result: string(data)})
	if err != nil {
		common.Log.Error(err.Error())
	}
}

//安装app
func (d *DeviceAgent) InstallApp(r *pb.InstallAppRequest, stream pb.DeviceAgentService_InstallAppServer) error {
	tmpdir := "tmp/" + utils.WDAControl.Uid
	filePath := TempFileName(tmpdir, ".ipa")
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
		if err := forceInstallAPK(filePath, r.Uid); err != nil {
			common.Log.Error(err.Error())
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
			common.Log.Error(err.Error())
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
			common.Log.Error(err.Error())
			return err
		}
		if resp.Status {
			if logSingle.logId == 0 {
				logSingle.logId = resp.LogId
			}
			if logSingle.status {
				//log已经启动
				common.Log.Warn("日志已经启动")
				err = stream.Send(&pb.LogStreamResponse{Data: "已经启动"})
				if err != nil {
					common.Log.Info(err.Error())
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
	snowId := utils.SnowId()
	uid := utils.WDAControl.Uid
	logIdStr := strconv.FormatInt(int64(r.LogId), 10)
	logPath := path.Join(`./`, `data`, uid, logIdStr)
	logKey := configs.ServerKey + snowId
	logSingle.name = path.Join(logPath, logKey+".log")
	if !common.PathExists(logPath) {
		os.MkdirAll(logPath, os.ModePerm)
	}
	mysql := common.GetMysql()
	newLog := model.FileModel{UserId: int(r.UserId), GroupId: int(r.GroupId),
		LogId: int(r.LogId), Name: logKey + ".log"}
	result := mysql.Create(&newLog)
	if result.Error != nil || result.RowsAffected != 1 {
		common.Log.Error(result.Error.Error(), zap.String("log", "日志记录失败"))
		logSingle.status = false
		err := stream.Send(&pb.LogStreamResponse{Data: "日志记录创建失败"})
		if err != nil {
			common.Log.Error(err.Error())
		}
		return
	}
	logSingle.Id = newLog.ID
	d.execLogShell()
	var logfile *tail.Tail
	defer func() {
		common.Log.Info("关闭日志文件")
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
			common.Log.Error(err.Error())
		}
		logSingle.status = false
		return
	}
	content := [configs.LogMsgLength]*logMsg{}
	logTimeStart := time.Now().Unix()
	logIndex := 0
	defer func() {
		common.Log.Info("尝试一下日志上传")
		//上传日志并回调日志服务，清理本地文件
		d.uploadLog(name)
	}()
loop:
	for {
		if logIndex == configs.LogMsgLength || time.Now().Unix()-logTimeStart > 1 {
			err = d.sendLog(content, stream)
			if err != nil {
				common.Log.Error(err.Error())
				d.stopLog(stream)
				break loop
			}
			content = [configs.LogMsgLength]*logMsg{}
			logTimeStart = time.Now().Unix()
			logIndex = 0
		}
		select {
		case _, ok := <-logSingle.stop:
			if !ok {
				logSingle.status = false
				common.Log.Info("释放日志")
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
func (d *DeviceAgent) sendLog(msg [configs.LogMsgLength]*logMsg, stream pb.DeviceAgentService_LogStreamServer) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		msgBytes = []byte{}
	}
	msgStr := string(msgBytes)
	err = stream.Send(&pb.LogStreamResponse{Data: msgStr})
	if err != nil {
		common.Log.Error(err.Error())
		return err
	}
	return nil
}
func (d *DeviceAgent) execLogShell() {
	cmdStr := fmt.Sprintf(`idevicesyslog -u %s >> %s`, utils.WDAControl.Uid, logSingle.name)
	logCmd := common.CmdService{
		Log:  common.Log,
		Name: "sh",
		Args: []string{"-c", cmdStr},
	}
	logCmd.Run()
	logSingle.status = true
	logSingle.cmd = logCmd

}
func (d *DeviceAgent) stopLog(stream pb.DeviceAgentService_LogStreamServer) {
	//关闭log，首先将log的进程杀掉
	common.Log.Info("我要关闭log了")
	if !logSingle.status {
		common.Log.Info("日志已经关闭")
		err := stream.Send(&pb.LogStreamResponse{Data: "日志读取失败"})
		if err != nil {
			common.Log.Error(err.Error())
		}
		return
	}
	err := syscall.Kill(-logSingle.cmd.Pid, syscall.SIGKILL)
	if err != nil {
		common.Log.Warn(err.Error(), zap.String("log", "杀日志进程出现问题"))
	}
	logSingle.status = false
	close(logSingle.stop)
}

func (d *DeviceAgent) uploadLog(josnPath string) {
	//上传日志文件
	logPath := strings.Replace(josnPath, ".json", ".log", 1)
	file := common.FileResult{}
	err := common.UploadFile(logPath, &file)
	if err != nil {
		common.Log.Error(err.Error())
	}
	db := common.GetMysql()
	size := strconv.FormatInt(file.Size, 10)
	result := db.Model(&model.FileModel{}).Where("id = ?", logSingle.Id).Updates(model.FileModel{Domain: file.Domain, Path: file.Path, Md5Id: file.Md5, Size: size, Type: "log"})
	common.Log.Info("我要修改日志记录了啊")
	if result.RowsAffected != 1 {
		common.Log.Error("日志添加失败" + utils.WDAControl.Uid)
		return
	}
	err = os.RemoveAll(path.Dir(josnPath))
	if err != nil {
		common.Log.Error(err.Error())
	}
}

func (d *DeviceAgent) RemoteDebug(ctx context.Context, r *pb.RemoteDebugRequest) (*pb.RemoteDebugResponse, error) {
	return &pb.RemoteDebugResponse{}, nil
}

func (d *DeviceAgent) ResetEnv(ctx context.Context, r *pb.ResetEnvRequest) (*pb.ResetEnvResponse, error) {
	return &pb.ResetEnvResponse{Status: true}, nil
}

func (d *DeviceAgent) VerifyCode(ctx context.Context, r *pb.VerifyCodeRequest) (*pb.VerifyCodeResponse, error) {
	var status = true
	if r.Code != utils.WDAControl.Code {
		status = false
		common.Log.Warn("非法访问", zap.String("code", r.Code), zap.String("key", utils.WDAControl.Code))
	}
	return &pb.VerifyCodeResponse{Status: status}, nil
}

func (d *DeviceAgent) Stop(ctx context.Context, r *pb.StopRequest) (*pb.StopResponse, error) {
	utils.WDAControl.Server.Stop()
	return &pb.StopResponse{}, nil

}
