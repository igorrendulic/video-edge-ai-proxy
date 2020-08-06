// Copyright 2020 Wearless Tech Inc All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpcapi

import (
	"context"
	"io"
	"strconv"
	"sync"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/go-redis/redis/v8"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcImageHandler struct {
	redisConn      *redis.Client
	deviceMap      sync.Map
	processManager *services.ProcessManager
}

// NewGrpcImageHandler returns main GRPC API handler
func NewGrpcImageHandler(processManager *services.ProcessManager) *grpcImageHandler {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // TODO: set redis password and use it here
		DB:       0,  // use default DB
	})
	return &grpcImageHandler{
		redisConn:      rdb,
		deviceMap:      sync.Map{},
		processManager: processManager,
	}
}

func (gih *grpcImageHandler) toUint64(object map[string]interface{}, field string) int64 {
	if val, ok := object[field]; ok {
		strVal := val.(string)
		w, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			g.Log.Error("Failed to convert width to int", err)
		}
		return w
	}
	return 0
}

// StartProxy - starts proxying the process stream video/audio to destination RTMP on Chrysalis cloud
func (gih *grpcImageHandler) StartProxy(ctx context.Context, req *pb.StartProxyRequest) (*pb.ProxyResponse, error) {
	deviceID := req.DeviceId
	storeVideo := req.StoreVideo

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

	valMap := make(map[string]interface{}, 0)
	valMap[models.RedisLastAccessQueryTimeKey] = time.Now().Unix() * 1000
	valMap[models.RedisProxyRTMPKey] = true
	valMap[models.RedisProxyStoreKey] = req.StoreVideo

	rErr := gih.redisConn.HSet(ctx, models.RedisLastAccessPrefix+deviceID, valMap).Err()
	if rErr != nil {
		g.Log.Error("failed to store startproxy value map to redis", rErr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if info.RTMPStreamStatus == nil {
		info.RTMPStreamStatus = &models.RTMPStreamStatus{}
	}
	info.RTMPStreamStatus.Storing = storeVideo
	info.RTMPStreamStatus.Streaming = true

	_, sErr := gih.processManager.UpdateProcessInfo(info)
	if sErr != nil {
		g.Log.Error("failed to update stream info", deviceID, sErr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	resp := &pb.ProxyResponse{
		DeviceId:    deviceID,
		ProxyStream: info.RTMPStreamStatus.Streaming,
		StoreVideo:  info.RTMPStreamStatus.Storing,
	}

	return resp, nil
}

// StopProxy stops the RTMP streaming to the Chrysalis cloud and by default sets storage to false
func (gih *grpcImageHandler) StopProxy(ctx context.Context, req *pb.StopProxyRequest) (*pb.ProxyResponse, error) {
	deviceID := req.DeviceId

	info, err := gih.processManager.Info(deviceID)
	if err != nil {
		g.Log.Error("failed to get deviceID info", err)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if info.RTMPStreamStatus == nil { // shoulnd't happen, but just in case
		info.RTMPStreamStatus = &models.RTMPStreamStatus{}
	}
	info.RTMPStreamStatus.Storing = false
	info.RTMPStreamStatus.Streaming = false

	valMap := make(map[string]interface{}, 0)
	valMap[models.RedisLastAccessQueryTimeKey] = time.Now().Unix() * 1000
	valMap[models.RedisProxyRTMPKey] = false
	valMap[models.RedisProxyStoreKey] = false

	rErr := gih.redisConn.HSet(ctx, models.RedisLastAccessPrefix+deviceID, valMap).Err()
	if rErr != nil {
		g.Log.Error("failed to update on stopProxy redis", deviceID, rErr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	_, uerr := gih.processManager.UpdateProcessInfo(info)
	if uerr != nil {
		g.Log.Error("failed to update process info", uerr)
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	resp := &pb.ProxyResponse{
		DeviceId:    deviceID,
		ProxyStream: info.RTMPStreamStatus.Streaming,
		StoreVideo:  info.RTMPStreamStatus.Storing,
	}
	return resp, nil
}

// ListStreams returns the list of all streams regardless of their status
func (gih *grpcImageHandler) ListStreams(req *pb.ListStreamRequest, stream pb.Image_ListStreamsServer) error {
	err := gih.processManager.ListStream(stream.Context(), func(process *models.StreamProcess) error {
		res := &pb.ListStream{
			Name:       process.Name,
			Dead:       process.State.Dead,
			Error:      process.State.Error,
			ExitCode:   int64(process.State.ExitCode),
			Oomkilled:  process.State.OOMKilled,
			Paused:     process.State.Paused,
			Pid:        int32(process.State.Pid),
			Restarting: process.State.Restarting,
			Running:    process.State.Running,
			Status:     process.Status,
		}
		if process.State.Health != nil {
			res.FailingStreak = int64(process.State.Health.FailingStreak)
			res.HealthStatus = process.State.Health.Status
		}

		err := stream.Send(res)
		if err != nil {
			g.Log.Error("failed to send process item", err)
			return err
		}
		g.Log.Info("sent process with name: ", process.Name)
		return nil
	})
	if err != nil {
		g.Log.Error("failed to retrieve processes stream", err)
	}
	return nil
}

func (gih *grpcImageHandler) VideoLatestImage(stream pb.Image_VideoLatestImageServer) error {

	clientDeadline := time.Now().Add(time.Duration(15) * time.Second)
	streamContext, streamCancel := context.WithDeadline(stream.Context(), clientDeadline)
	defer streamCancel()

	for {

		select {
		case <-streamContext.Done():
			g.Log.Warn("Context done: ", streamContext.Err())
			return streamContext.Err()
		default:
		}

		request, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			g.Log.Error("failed to retrieve gprc image request", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()

		streamName := request.DeviceId
		isKeyFrameOnly := request.KeyFrameOnly

		decodeOnlyKeyFramesKey := models.RedisIsKeyFrameOnlyPrefix + streamName
		err = gih.redisConn.Set(ctx, decodeOnlyKeyFramesKey, strconv.FormatBool(isKeyFrameOnly), 0).Err()
		if err != nil {
			g.Log.Error("failed to set if is keyframe only", streamName, err)
		}

		// set last access time for the image on this device (streamName)
		val := time.Now().UnixNano() / int64(time.Millisecond)

		valMap := make(map[string]interface{}, 0)
		valMap[models.RedisLastAccessQueryTimeKey] = val

		rErr := gih.redisConn.HSet(ctx, models.RedisLastAccessPrefix+streamName, valMap).Err()
		if rErr != nil {
			g.Log.Error("failed to update on stopProxy redis", streamName, rErr)
			continue
		}

		// loading VideoFrame from redis
		vf := &pb.VideoFrame{}

		// check where we left off for device/streamName
		lastTs := "0"
		if last_ts, ok := gih.deviceMap.Load(streamName); ok {
			lastTs = last_ts.(string)
		}

		// waiting up to 20 x 40ms = 800ms seconds max to get an image from redis, otherwise considered there is none
		for i := 0; i < 10; i++ {

			imgFound := false

			args := &redis.XReadArgs{
				Streams: []string{streamName, lastTs},
				Block:   time.Second,
				Count:   60,
			}

			vals, err := gih.redisConn.XRead(ctx, args).Result()
			if err != nil {
				g.Log.Info("waiting for an image, retry #", i, err.Error())
				continue
			}
			for _, val := range vals {
				messages := val.Messages

				if len(messages) > 0 {
					// reading only latest image
					lastMessage := messages[len(messages)-1]
					id := lastMessage.ID
					gih.deviceMap.Store(streamName, id)

					object := lastMessage.Values
					if val, ok := object["data"]; ok {
						str := val.(string)
						b := []byte(str)
						err := proto.Unmarshal(b, vf)
						if err != nil {
							g.Log.Error("failed to unmarshall VideoFrame proto", err)
							break
						}
						imgFound = true
					}
				}
			}
			if imgFound {
				break
			}
			// delay or 40 ms before new attempty
			time.Sleep(time.Millisecond * 40)
		}

		if errStr := stream.Send(vf); errStr != nil {
			g.Log.Error("send error", errStr)
		}
	}
}
