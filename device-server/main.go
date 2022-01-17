/*
* @Author: 于智明
* @Date:   2021/2/3 3:48 下午
 */
package main

import (
	"ctp-device-server/configs"
	"ctp-device-server/global"
	cloudLog "ctp-device-server/moudles/log"
	middleware "ctp-device-server/moudles/middlerwaer"
	"ctp-device-server/moudles/modle"
	"fmt"
	mapSet "github.com/deckarep/golang-set"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	configs.Init()
	cloudLog.Logger.Info("", zap.Bool("是否为mac系统", configs.CloudConfig.Server.IsIos))
	model.Init()
	sysType := runtime.GOOS
	configs.CloudConfig.Server.IsIos = false
	if sysType == "darwin" {
		configs.CloudConfig.Server.IsIos = true
	}
	onlineDevices := mapSet.NewSet()
	go clearProcess(sysType)
	device := Device{
		OnlineDevices: onlineDevices,
		DB:            global.MysqlDb,
		RedisClient:   global.RedisDb,
	}
	go device.CheckDevice()
	err := rpcServer()
	fmt.Println(err)
	//todo = 有连接情况下主站重启或者断开连接
}

func rpcServer() error {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(middleware.AccessLog, middleware.ErrorLog)),
	}
	server := grpc.NewServer(opts...)
	registerRpc(server)
	lis, err := net.Listen("tcp", "0.0.0.0:"+configs.CloudConfig.Server.RpcPort)
	if err != nil {
		return err
	}
	return server.Serve(lis)

}

func clearProcess(sysType string) {
	for {
		time.Sleep(500 * time.Millisecond)
		iproxyCmd := exec.Command("sh", "-c", "ps -ef|grep iproxy")
		outputByte, _ := iproxyCmd.CombinedOutput()
		output := string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
		iosVideoCmd := exec.Command("sh", "-c", "ps -ef|grep ios-proxy-video")
		outputByte, _ = iosVideoCmd.CombinedOutput()
		output = string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
		xcodeCmd := exec.Command("sh", "-c", "ps -ef|grep xcodebuild")
		outputByte, _ = xcodeCmd.CombinedOutput()
		output = string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
		logCmd := exec.Command("sh", "-c", "ps -ef|grep idevicesyslog")
		outputByte, _ = logCmd.CombinedOutput()
		output = string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
		logAndroidCmd := exec.Command("sh", "-c", "ps -ef|grep logcat")
		outputByte, _ = logAndroidCmd.CombinedOutput()
		output = string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
		adbKitCmd := exec.Command("sh", "-c", "ps -ef|grep adbkit")
		outputByte, _ = adbKitCmd.CombinedOutput()
		output = string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
		//android-proxy-server
		androidVideoCmd := exec.Command("sh", "-c", "ps -ef|grep android-proxy-server")
		outputByte, _ = androidVideoCmd.CombinedOutput()
		output = string(outputByte)
		if output != "" {
			releaseIproxy(output, sysType)
		}
	}

}
func releaseIproxy(output string, sysType string) {
	info := strings.Split(output, "\n")

	for _, v := range info {
		if v == "" {
			return
		}
		var pid string
		var ppid string
		if len(v) < 20 {
			continue
		}
		if sysType == "darwin" {
			pid = strings.TrimSpace(v[6:11])
			ppid = strings.TrimSpace(v[12:17])
		} else {
			pid = strings.TrimSpace(v[9:14])
			ppid = strings.TrimSpace(v[15:20])
		}
		if ppid == "1" {
			pidInt, err := strconv.Atoi(pid)
			if err != nil {
				continue
			}
			cloudLog.Logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>")
			cloudLog.Logger.Info(v, zap.String("msg", "清理进程"))
			cloudLog.Logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>")
			err = syscall.Kill(pidInt, syscall.SIGKILL)
			if err != nil {
				cloudLog.Logger.Warn(err.Error())
			}
		}
	}
}
