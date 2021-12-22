package main

import (
	"context"
	"flag"
	"fmt"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"ctp-ios-proxy/common"
	"ctp-ios-proxy/configs"
	"ctp-ios-proxy/middleware"
	"ctp-ios-proxy/utils"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
)

func main() {
	var rpcPort string
	var uid string
	var code string
	flag.StringVar(&rpcPort, "rpc_port", "18", "rpc启动端口")
	flag.StringVar(&uid, "uid", "18", "设备序列号")
	flag.StringVar(&code, "code", "18", "code")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if rpcPort == "18" || uid == "18" || code == "18" {
		panic("rpcPort uid ")
	}
	common.Log = common.GetLog(uid)
	common.InItMysql()
	agentPort := strconv.Itoa(common.GetFreePort())
	controlPort := strconv.Itoa(common.GetFreePort())
	common.Log.Info("rpcPort:" + rpcPort)
	common.Log.Info("9100端口转发:" + agentPort)
	common.Log.Info("8100端口转发:" + controlPort)
	utils.WDAControl = utils.WDA{
		AgentPort:   agentPort,
		ControlPort: controlPort,
		Uid:         uid,
		CmdList:     []common.CmdUtil{},
		Status:      configs.StatusWait,
		Code:        code,
	}
	common.Log.Info(configs.ConfigiOS.Server.Iproxy)
	var wg sync.WaitGroup
	wg.Add(1)
	go utils.WDAControl.Start(ctx, &wg)
	os.RemoveAll("data/" + uid)
	os.RemoveAll("tmp/" + uid)
	defer func() {
		utils.WDAControl.Stop()
		common.Log.Sync()
	}()
	err := rpcServer(rpcPort, cancel)
	if err != nil {
		panic(err)
	}
	wg.Wait()
}

func rpcServer(rpcPort string, cancel context.CancelFunc) error {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(middleware.AccessLog, middleware.ErrorLog)),
		grpc.MaxSendMsgSize(math.MaxUint32),
		grpc.MaxRecvMsgSize(math.MaxUint32),
	}
	server := grpc.NewServer(opts...)
	registerRpc(server)
	fmt.Println("rpc启动端口" + rpcPort)
	lis, err := net.Listen("tcp", ":"+rpcPort)
	if err != nil {
		return err
	}
	utils.WDAControl.Server = server
	err = server.Serve(lis)
	cancel()
	return err

}
