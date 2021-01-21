package grpcapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/go-resty/resty/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Storage - enable disable storage on Chrysalis Cloud
func (gih *grpcImageHandler) Storage(ctx context.Context, req *pb.StorageRequest) (*pb.StorageResponse, error) {

	deviceID := req.DeviceId

	if deviceID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device id required")
	}

	info, err := gih.processManager.Info(deviceID)
	if err != nil {
		g.Log.Error("failed to get deviceID info", err)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if info.RTMPEndpoint == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device "+deviceID+" doesn't have an associated RTMP stream")
	}

	apiErr := gih.enableDisableStorageAPICall(req.Start, info.RTMPEndpoint)
	if apiErr != nil {
		if apiErr == models.ErrForbidden {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("cannot enable or disable storage on chrysalis cloud: %v", apiErr.Error()))
	}

	if info.RTMPStreamStatus == nil {
		info.RTMPStreamStatus = &models.RTMPStreamStatus{}
	}
	info.RTMPStreamStatus.Storing = req.Start
	info.Modified = time.Now().Unix() * 1000

	_, sErr := gih.processManager.UpdateProcessInfo(info)
	if sErr != nil {
		g.Log.Error("failed to update stream info", deviceID, sErr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	resp := &pb.StorageResponse{
		DeviceId: deviceID,
		Start:    req.Start,
	}
	return resp, nil
}

// API call to enable or disable storage on chrysalis cloud
func (gih *grpcImageHandler) enableDisableStorageAPICall(storageOn bool, rtmpEndpoint string) error {
	key, rtmpErr := utils.ParseRTMPKey(rtmpEndpoint)
	if rtmpErr != nil {
		g.Log.Error("Failed to parse rtmp key from rtmp url", rtmpErr)
		return rtmpErr
	}
	input := &StorageInput{
		Enable: storageOn,
	}
	if g.Conf.API.Endpoint == "" {
		return errors.New("missing Chrysalis Cloud API endpoint in settings")
	}

	edgeKey, edgeSecret, eErr := gih.settingsManager.GetCurrentEdgeKeyAndSecret()
	if eErr != nil {
		g.Log.Error("Can't find edge key and secret. Visit https://cloud.chryscloud.com to enable annotation and storage.", eErr)
		return errors.New("Can't find edge key and secret. Visit https://cloud.chryscloud.com to enable annotation and storage.")
	}

	apiClient := resty.New()
	_, apiErr := utils.CallAPIWithBody(apiClient, "PUT", g.Conf.API.Endpoint+"/api/v1/edge/storage/"+key, input, edgeKey, edgeSecret)
	if apiErr != nil {
		g.Log.Error("failed to call Chrysalis Cloud API: ", apiErr)
		return apiErr
	}
	return nil
}
