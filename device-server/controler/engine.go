/*
* @Author: 于智明
* @Date:   2021/2/4 2:27 下午
 */
package controler

import (
	"context"
	"ctp-device-server/configs"
	cloudLog "ctp-device-server/moudles/log"
	"ctp-device-server/moudles/shell"
	"ctp-device-server/moudles/tools"
	pb "ctp-device-server/proto"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"strconv"
	"time"
)

type Device struct {
	Uid  string
	port string
}

func (i *Device) Stop() bool {

	return true
}

func (i *Device) Start(code string) bool {
	var agentPath string
	if configs.CloudConfig.Server.IsIos {
		agentPath = "./ios-proxy-video"
	} else {
		agentPath = "./android-proxy-server"
	}
	freePort := strconv.Itoa(tools.GetFreePort())
	cloudLog.Logger.Info("启动端口"+freePort, zap.String("uid", i.Uid))
	i.port = freePort
	args := []string{"--uid=" + i.Uid, "--rpc_port=" + freePort, "--code=" + code}
	cmd := shell.Service{Name: agentPath, Args: args}
	cmd.Start()
	if !cmd.Success {
		return false
	}
	for m := 0; m < 30; m++ {
		time.Sleep(time.Second)
		flag, err := i.Ping(freePort)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		switch flag.Status {
		case pb.PingResponse_Fail:
			return false
		case pb.PingResponse_Success:
			cloudLog.Logger.Info(flag.ControlPort)
			cloudLog.Logger.Info(flag.VideoPort)
			return true
		case pb.PingResponse_Wait:
			continue
		}
	}

	return false
}

func (i *Device) Ping(port string) (*pb.PingResponse, error) {
	cloudLog.Logger.Info("访问端口" + port)
	conn, err := grpc.Dial(":"+port, grpc.WithInsecure())
	defer func() {
		conn.Close()
	}()
	if err != nil {
		cloudLog.Logger.Error(err.Error())
	}
	client := pb.NewDeviceAgentServiceClient(conn)
	data, err := client.Ping(context.Background(), &pb.PingRequest{})
	if err != nil {
		cloudLog.Logger.Error(err.Error())
		return &pb.PingResponse{}, err
	}
	return data, nil

}

func (i *Device) Port() string {
	return i.port
}
