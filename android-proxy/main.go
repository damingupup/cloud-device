package main

import (
	"context"
	"ctp-android-proxy/global"
	cloudlog "ctp-android-proxy/moudles/log"
	"ctp-android-proxy/moudles/middleware"
	model "ctp-android-proxy/moudles/modle"
	"ctp-android-proxy/moudles/tools"
	"flag"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	flag.StringVar(&uid, "uid", "18", "uid")
	flag.StringVar(&code, "code", "18", "code")
	flag.Parse()
	cloudlog.Init(uid)
	workdir, _ := os.Getwd()
	cloudlog.Logger.Info("*******", zap.String("工作目录", workdir))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cloudlog.Logger.Sync()
	}()
	if rpcPort == "18" || uid == "18" || code == "18" {
		panic("rpc_port SecretKey ")
	}
	agentPort := strconv.Itoa(tools.GetFreePort())
	model.Init()
	os.RemoveAll("data/" + uid)
	os.RemoveAll("tmp/" + uid)
	tools.AndroidControl = tools.AndroidEngine{
		AgentPort:   agentPort,
		ControlPort: agentPort,
		Uid:         uid,
		Status:      global.StatusWait,
		RemoteDebug: nil,
		Code:        code,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go tools.AndroidControl.Start(ctx, &wg)
	err := rpcServer(cancel, rpcPort, &wg)
	if err != nil {
		panic(err)
	}
	wg.Wait()
}

func rpcServer(cancel context.CancelFunc, rpcPort string, wg *sync.WaitGroup) (err error) {
	wg.Add(1)
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(middleware.AccessLog, middleware.ErrorLog)),
		grpc.MaxSendMsgSize(256 * 1024 * 1024),
	}
	server := grpc.NewServer(opts...)
	registerRpc(server)
	tools.AndroidControl.Server = server
	lis, err := net.Listen("tcp", "0.0.0.0:"+rpcPort)
	if err != nil {
		return err
	}
	err = server.Serve(lis)
	cloudlog.Logger.Error("程序退出")
	cancel()
	wg.Done()
	return err

}
