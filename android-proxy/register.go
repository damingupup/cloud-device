package main

import (
	"ctp-android-proxy/controler"
	pb "ctp-android-proxy/proto"
	"google.golang.org/grpc"
)

func registerRpc(server *grpc.Server) {
	pb.RegisterDeviceAgentServiceServer(server, &controler.DeviceAgent{})
	//reflection.Register(server)
}
