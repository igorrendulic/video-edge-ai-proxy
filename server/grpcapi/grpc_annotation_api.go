package grpcapi

import (
	"context"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (gih *grpcImageHandler) Annotate(ctx context.Context, req *pb.AnnotateRequest) (*pb.AnnotateResponse, error) {
	if gih.edgeKey == nil {
		settings, err := gih.settingsManager.Get()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to read settings")
		}
		if settings.EdgeKey == "" {
			return nil, status.Errorf(codes.InvalidArgument, "can't find edge key in settings. Required to use annotations")
		}
		gih.edgeKey = &settings.EdgeKey
	}
	if req.DeviceName == "" || req.Type == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device_name and type (event type) required")
	}

	edgeKey := *gih.edgeKey
	if edgeKey == "" {
		g.Log.Info("WTF>")
	}

	batchErr := gih.batching.Add(req)
	if batchErr != nil {
		g.Log.Error("failed to process event batch", batchErr)
		return nil, status.Errorf(codes.Internal, batchErr.Error())
	}

	resp := &pb.AnnotateResponse{
		DeviceName:     req.DeviceName,
		StartTimestamp: time.Now().Unix() * 1000,
		Type:           req.Type,
	}
	return resp, nil
}
