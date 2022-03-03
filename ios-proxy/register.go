/*
* @Author: 于智明
* @Date:   2021/1/12 7:59 下午
 */
package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"ios-proxy/controler"
	pb "ios-proxy/proto"
)

func registerRpc(server *grpc.Server) {
	pb.RegisterDeviceAgentServiceServer(server, &controler.DeviceAgent{})
	reflection.Register(server)
}
