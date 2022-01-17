/*
* @Author: 于智明
* @Date:   2021/2/4 11:15 上午
 */
package controler

import (
	"context"
	"ctp-device-server/configs"
	"ctp-device-server/global"
	cloudLog "ctp-device-server/moudles/log"
	model "ctp-device-server/moudles/modle"
	"ctp-device-server/moudles/rpcutils"
	"ctp-device-server/moudles/shell"
	"ctp-device-server/moudles/tools"
	pb "ctp-device-server/proto"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type DeviceServer struct {
}

func (d *DeviceServer) Connect(ctx context.Context, r *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	var device Device
	device = Device{Uid: r.SerId}
	code := rpcutils.SnowId()
	flag := device.Start(code)
	return &pb.ConnectResponse{Status: flag, Port: device.Port(), Code: code}, nil
}

const (
	isStart = true
	isStop  = false
)

type videoEngine struct {
	Uid     string
	Port    int32
	LogId   string
	SnowId  string
	Id      int
	Request *pb.SaveVideoClientRequest
}

func (d *DeviceServer) SaveVideo(ctx context.Context, r *pb.SaveVideoClientRequest) (*pb.SaveVideoClientResponse, error) {
	logIdStr := strconv.FormatInt(int64(r.LogId), 10)
	snowId := rpcutils.SnowId()
	video := videoEngine{Uid: r.Uid, Port: r.Port, LogId: logIdStr, SnowId: snowId}
	if r.Status {
		basePath := path.Join("tmp", r.Uid, logIdStr)
		os.MkdirAll(basePath, os.ModePerm)
		startPath := path.Join("tmp", r.Uid, logIdStr, "start.start")
		startFlag := video.judgeStatus(isStart)
		if startFlag {
			cloudLog.Logger.Warn("视频录制已经开启")
			return &pb.SaveVideoClientResponse{Msg: "视频录制已经开启"}, nil
		}
		_, err := os.Create(startPath)
		if err != nil {
			cloudLog.Logger.Error(err.Error())
		}
		cloudLog.Logger.Warn("视频录制开启")
		go video.startSaveVideo(r)
		return &pb.SaveVideoClientResponse{}, nil
	} else {
		stopFlag := video.judgeStatus(isStop)
		if stopFlag {
			cloudLog.Logger.Warn("视频已经关闭")
			return &pb.SaveVideoClientResponse{Msg: "视频已经关闭"}, nil
		}
		cloudLog.Logger.Warn("视频关闭")
		video.stopSaveVideo()
		return &pb.SaveVideoClientResponse{}, nil
	}

}
func (v *videoEngine) startSaveVideo(r *pb.SaveVideoClientRequest) {
	db := global.MysqlDb
	//创建视频记录
	newVideo := model.FileModel{UserId: int(r.UserId), GroupId: int(r.GroupId),
		LogId: int(r.LogId), Name: v.SnowId + ".mp4", Type: "mp4"}
	v.Request = r
	result := db.Create(&newVideo)
	if result.Error != nil || result.RowsAffected != 1 {
		cloudLog.Logger.Error(result.Error.Error(), zap.String("log", "视频记录失败"))
		return
	}
	v.Id = int(newVideo.ID)
	//连接视频流端口
	port := strconv.FormatInt(int64(v.Port), 10)
	conn, err := grpc.Dial(":"+port, grpc.WithInsecure())
	defer func() {
		fmt.Println("关闭")
		conn.Close()
	}()
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
	client := pb.NewDeviceAgentServiceClient(conn)
	stream, err := client.VideoStream(context.Background(), &pb.VideoStreamRequest{})
	if err != nil {
		cloudLog.Logger.Error(err.Error())
		return
	}
	//是否有封面图
	isFirst := false
	picPath := path.Join("tmp", v.Uid, v.LogId, v.SnowId)
	os.MkdirAll(picPath, os.ModePerm)
	flag := false
	go v.updateStatus(&flag)
	ch := make(chan bool, 20)
	var wg = sync.WaitGroup{}
	for {
		data, err := stream.Recv()
		if err != nil {
			v.stopSaveVideo()
			v.convertVideo()
			break
		}
		if !isFirst {
			isFirst = true
			v.saveFirst(data.PicBytes)
		}
		if flag {
			//发现停止文件
			v.convertVideo()
			wg.Wait()
			break
		}
		go v.savePicture(data.PicBytes, ch, &wg)
	}
}
func (v *videoEngine) saveFirst(data []byte) {
	name := v.SnowId + ".jpg"
	basePath := path.Join("tmp", v.Uid, v.LogId)
	firstPath := path.Join(basePath, name)
	err := ioutil.WriteFile(firstPath, data, os.ModePerm)
	if err != nil {
		cloudLog.Logger.Error(err.Error())
		return
	}
	db := global.MysqlDb
	newPic := model.FileModel{Type: "jpg", Name: name, FiledId: v.Id, UserId: int(v.Request.UserId),
		GroupId: int(v.Request.GroupId), LogId: int(v.Request.LogId)}
	result := db.Create(&newPic)
	if result.Error != nil {
		cloudLog.Logger.Error(result.Error.Error(), zap.String("log", "封面记录失败"))
		return
	}
	file := tools.FileResult{}
	err = tools.UploadFile(firstPath, &file)
	if err != nil {
		cloudLog.Logger.Error(err.Error(), zap.String("pic", "封面图获取失败"))
		return
	}
	size := strconv.FormatInt(file.Size, 10)
	result = db.Model(&newPic).Updates(model.FileModel{Domain: file.Domain, Path: file.Path, Md5Id: file.Md5, Size: size})
	if result.RowsAffected != 1 {
		cloudLog.Logger.Error("封面添加失败" + v.Uid)
		return
	}
}
func (v *videoEngine) savePicture(data []byte, ch chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	ch <- true
	name := strconv.FormatInt((time.Now().UnixNano())/1000, 10) + ".jpg"
	basePath := path.Join("tmp", v.Uid, v.LogId, v.SnowId)
	filePath := path.Join(basePath, name)
	err := ioutil.WriteFile(filePath, data, 0600)
	if err != nil {
		cloudLog.Logger.Error(err.Error())
		return
	}
	<-ch
}

func (v *videoEngine) stopSaveVideo() {
	basePath := path.Join("tmp", v.Uid, v.LogId)
	stopPath := path.Join(basePath, "stop.stop")
	os.Create(stopPath)
}
func (v *videoEngine) convertVideo() {
	v.clearSymbol()
	args := []string{"convert.py", v.Uid, v.LogId, v.SnowId}
	cmd := shell.Service{Name: configs.CloudConfig.Server.Python, Args: args,
		Show: true}
	cmd.Run()
	videoDir := path.Join("tmp", v.Uid, v.LogId, v.SnowId)
	videoPath := path.Join("tmp", v.Uid, v.LogId, v.SnowId+".mp4")
	defer func() {
		err := os.RemoveAll(videoDir)
		if err != nil {
			cloudLog.Logger.Warn(err.Error())
		}
		err = os.Remove(videoPath)
		if err != nil {
			cloudLog.Logger.Warn(err.Error())
		}
		firstPath := path.Join("tmp", v.Uid, v.LogId, v.SnowId+".jpg")
		err = os.Remove(firstPath)
		if err != nil {
			cloudLog.Logger.Warn(err.Error())
		}
	}()
	file := tools.FileResult{}
	err := tools.UploadFile(videoPath, &file)
	if err != nil {
		cloudLog.Logger.Error(err.Error(), zap.String("pic", "视频上传失败"))
		return
	}
	db := global.MysqlDb
	size := strconv.FormatInt(file.Size, 10)
	result := db.Model(&model.FileModel{}).Where("id=?", v.Id).Updates(model.FileModel{Domain: file.Domain, Path: file.Path, Md5Id: file.Md5, Size: size})
	if result.RowsAffected != 1 {
		cloudLog.Logger.Error("封面添加失败" + v.Uid)
	}
}

func (v videoEngine) updateStatus(flag *bool) {
	for {
		*flag = v.judgeStatus(isStop)
		time.Sleep(time.Second)
	}
}
func (v videoEngine) clearSymbol() {
	basePath := path.Join("tmp", v.Uid, v.LogId)
	startPath := path.Join(basePath, "start.start")
	stopPath := path.Join(basePath, "stop.stop")
	os.Remove(startPath)
	os.Remove(stopPath)
}

func (v *videoEngine) judgeStatus(stat bool) bool {
	var filePath string
	if stat {
		//判断开始标志文件路径
		filePath = path.Join("tmp", v.Uid, v.LogId, "start.start")
	} else {
		//判断结束标志文件路径
		filePath = path.Join("tmp", v.Uid, v.LogId, "stop.stop")
	}
	_, err := os.Stat(filePath)
	if err == nil {
		return true
	} else {
		if os.IsExist(err) {
			return true
		}
		return false
	}

}
