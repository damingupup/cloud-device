package middleware

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"ios-proxy/common"
	"ios-proxy/utils"
	"time"
)

func AccessLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	requestLog := "access request log: method: %s, begin_time: %d, request: %v"
	beginTime := time.Now().Local().Unix()
	msg := fmt.Sprintf(requestLog, info.FullMethod, beginTime, req)
	common.Log.Info(msg)
	resp, err := handler(ctx, req)
	responseLog := "access response log: method: %s, begin_time: %d, end_time: %d, response: %v"
	endTime := time.Now().Local().Unix()
	msg = fmt.Sprintf(responseLog, info.FullMethod, beginTime, endTime, resp)
	common.Log.Info(msg)
	return resp, err
}

func ErrorLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		errLog := "error log: method: %s, code: %v, message: %v, details: %v"
		s := utils.FromError(err)
		msg := fmt.Sprintf(errLog, info.FullMethod, s.Code(), s.Err().Error(), s.Details())
		common.Log.Error(msg)
	}
	return resp, err
}
