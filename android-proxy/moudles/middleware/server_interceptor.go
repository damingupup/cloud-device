package middleware

import (
	"context"
	cloudlog "ctp-android-proxy/moudles/log"
	"ctp-android-proxy/moudles/rpcerror"
	"fmt"
	"google.golang.org/grpc"
	"time"
)

func AccessLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	requestLog := "access request log: method: %s, begin_time: %d, request: %v"
	beginTime := time.Now().Local().Unix()
	msg := fmt.Sprintf(requestLog, info.FullMethod, beginTime, req)
	cloudlog.Logger.Info(msg)
	resp, err := handler(ctx, req)
	responseLog := "access response log: method: %s, begin_time: %d, end_time: %d, response: %v"
	endTime := time.Now().Local().Unix()
	msg = fmt.Sprintf(responseLog, info.FullMethod, beginTime, endTime, resp)
	cloudlog.Logger.Info(msg)
	return resp, err
}

func ErrorLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		errLog := "error log: method: %s, code: %v, message: %v, details: %v"
		s := rpcerror.FromError(err)
		msg := fmt.Sprintf(errLog, info.FullMethod, s.Code(), s.Err().Error(), s.Details())
		cloudlog.Logger.Error(msg)
	}
	return resp, err
}
