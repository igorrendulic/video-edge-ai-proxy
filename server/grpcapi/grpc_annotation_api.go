package grpcapi

import (
	"context"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Annotate queues a new annotation event to be sent to Chrysalis event servers
func (gih *grpcImageHandler) Annotate(ctx context.Context, req *pb.AnnotateRequest) (*pb.AnnotateResponse, error) {
	if gih.edgeKey == nil {
		settings, err := gih.settingsManager.Get()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to read settings")
		}
		if settings.EdgeKey == "" {
			return nil, status.Errorf(codes.InvalidArgument, "Can't find edge key in settings. required to use annotations. Visit https://cloud.chryscloud.com to enable annotations and storage capabilities from the edge.")
		}
		gih.edgeKey = &settings.EdgeKey
	}
	weekPast := time.Now().AddDate(0, 0, -7).Unix() * 1000
	weekFuture := time.Now().AddDate(0, 0, 7).Unix() * 1000
	if req.DeviceName == "" || req.Type == "" || req.StartTimestamp < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device_name and type (event type) required")
	}
	if req.StartTimestamp < weekPast || req.StartTimestamp > weekFuture {
		return nil, status.Errorf(codes.InvalidArgument, "start_timestamp must not be older than 7 days and not more than 7 days in the future")
	}

	edgeKey := *gih.edgeKey
	if edgeKey == "" {
		g.Log.Info("WTF>")
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		g.Log.Error("invalid proto format for annotation", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid annotation proto format")
	}
	ok := gih.msgQueue.PublishBytes(reqBytes)
	if !ok {
		g.Log.Error("failed to publish to msg queue", ok)
		return nil, status.Errorf(codes.Internal, "failed to publish to msg queue")
	}

	resp := &pb.AnnotateResponse{
		DeviceName:     req.DeviceName,
		StartTimestamp: req.StartTimestamp,
		Type:           req.Type,
	}
	return resp, nil
}
