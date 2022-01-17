package middleware

import (
	"context"
	"ctp-device-server/moudles/log"
	rpcUtils "ctp-device-server/moudles/rpcutils"
	"fmt"
	"google.golang.org/grpc"
	"time"
)

func AccessLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	requestLog := "access request log: method: %s, begin_time: %s, request: %v"
	beginTime := time.Now().Local().String()
	msg := fmt.Sprintf(requestLog, info.FullMethod, beginTime, req)
	cloudlog.Logger.Info(msg)
	resp, err := handler(ctx, req)
	responseLog := "access response log: method: %s, begin_time: %s, end_time: %s, response: %v"
	endTime := time.Now().Local().String()
	msg = fmt.Sprintf(responseLog, info.FullMethod, beginTime, endTime, resp)
	cloudlog.Logger.Info(msg)
	return resp, err
}

func ErrorLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		errLog := "error log: method: %s, code: %v, message: %v, details: %v"
		s := rpcUtils.FromError(err)
		msg := fmt.Sprintf(errLog, info.FullMethod, s.Code(), s.Err().Error(), s.Details())
		cloudlog.Logger.Error(msg)
	}
	return resp, err
}
