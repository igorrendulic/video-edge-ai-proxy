package grpcapi

import (
	"context"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (gih *grpcImageHandler) Proxy(ctx context.Context, req *pb.ProxyRequest) (*pb.ProxyResponse, error) {
	deviceID := req.DeviceId

	if deviceID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device id required")
	}

	info, err := gih.processManager.Info(deviceID)
	if err != nil {
		g.Log.Error("failed to get deviceID info", err)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if info.RTMPEndpoint == "" && req.Passthrough {
		return nil, status.Errorf(codes.InvalidArgument, "device "+deviceID+" doesn't have an associated RTMP stream. Visit https://cloud.chryscloud.com and add a RTMP stream.")
	}

	valMap := make(map[string]interface{}, 0)
	valMap[models.RedisLastAccessQueryTimeKey] = time.Now().Unix() * 1000
	valMap[models.RedisProxyRTMPKey] = req.Passthrough

	rErr := gih.redisConn.HSet(models.RedisLastAccessPrefix+deviceID, valMap).Err()
	if rErr != nil {
		g.Log.Error("failed to store startproxy value map to redis", rErr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if info.RTMPStreamStatus == nil {
		info.RTMPStreamStatus = &models.RTMPStreamStatus{}
	}
	info.RTMPStreamStatus.Streaming = req.Passthrough

	_, sErr := gih.processManager.UpdateProcessInfo(info)
	if sErr != nil {
		g.Log.Error("failed to update stream info", deviceID, sErr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	resp := &pb.ProxyResponse{
		DeviceId:    deviceID,
		Passthrough: info.RTMPStreamStatus.Streaming,
	}

	return resp, nil
}
