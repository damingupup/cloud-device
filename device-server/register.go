/*
* @Author: 于智明
* @Date:   2021/1/12 7:59 下午
 */
package main

import (
	"ctp-device-server/controler"
	pb "ctp-device-server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func registerRpc(server *grpc.Server) {
	pb.RegisterInstallAppServer(server, &controler.AppHandlerInstallRpc{})
	pb.RegisterDeviceServiceServer(server, &controler.DeviceServer{})
	reflection.Register(server)
}
