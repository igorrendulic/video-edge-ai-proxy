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

	"github.com/adjust/rmq/v2"
	"github.com/chryscloud/video-edge-ai-proxy/batch"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/go-redis/redis/v7"
	"github.com/golang/protobuf/proto"
)

type StorageInput struct {
	Enable bool `json:"enable"`
}

type grpcImageHandler struct {
	redisConn       *redis.Client
	deviceMap       sync.Map
	processManager  *services.ProcessManager
	settingsManager *services.SettingsManager
	edgeService     *services.EdgeService
	edgeKey         *string
	msgQueue        rmq.Queue
}

// NewGrpcImageHandler returns main GRPC API handler
func NewGrpcImageHandler(processManager *services.ProcessManager, settingsManager *services.SettingsManager, edgeService *services.EdgeService, rdb *redis.Client) *grpcImageHandler {

	// var rdb *redis.Client
	// for i := 0; i < 3; i++ {
	// 	rdb = redis.NewClient(&redis.Options{
	// 		Addr:     g.Conf.Redis.Connection,
	// 		Password: g.Conf.Redis.Password,
	// 		DB:       g.Conf.Redis.Database,
	// 	})

	// 	status := rdb.Ping()
	// 	g.Log.Info("redis status: ", status)
	// 	if status.Err() != nil {
	// 		g.Log.Warn("waiting for redis to boot up", status.Err().Error)
	// 		time.Sleep(3 * time.Second)
	// 		continue
	// 	}
	// 	break
	// }

	conn := rmq.OpenConnectionWithRedisClient("annotationService", rdb)
	msgQueue := conn.OpenQueue("annotationqueue")

	// add batch listener (consumer) for annotatons
	annotationConsumer := batch.NewAnnotationConsumer(0, settingsManager, edgeService, msgQueue)
	msgQueue.StartConsuming(g.Conf.Annotation.UnackedLimit, time.Duration(g.Conf.Annotation.PollDurationMs)*time.Millisecond)
	msgQueue.AddBatchConsumerWithTimeout("annotationqueue", g.Conf.Annotation.MaxBatchSize, time.Duration(g.Conf.Annotation.PollDurationMs)*time.Millisecond, annotationConsumer)

	return &grpcImageHandler{
		redisConn:       rdb,
		deviceMap:       sync.Map{},
		processManager:  processManager,
		settingsManager: settingsManager,
		edgeService:     edgeService,
		msgQueue:        msgQueue,
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

		streamName := request.DeviceId
		isKeyFrameOnly := request.KeyFrameOnly

		decodeOnlyKeyFramesKey := models.RedisIsKeyFrameOnlyPrefix + streamName
		err = gih.redisConn.Set(decodeOnlyKeyFramesKey, strconv.FormatBool(isKeyFrameOnly), 0).Err()
		if err != nil {
			g.Log.Error("failed to set if is keyframe only", streamName, err)
		}

		// set last access time for the image on this device (streamName)
		val := time.Now().UnixNano() / int64(time.Millisecond)

		valMap := make(map[string]interface{}, 0)
		valMap[models.RedisLastAccessQueryTimeKey] = val

		rErr := gih.redisConn.HSet(models.RedisLastAccessPrefix+streamName, valMap).Err()
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

		// waiting up to 3 x 16ms = max to get an image from redis, otherwise considered that there is no image
		for i := 0; i < 3; i++ {

			imgFound := false

			args := &redis.XReadArgs{
				Streams: []string{streamName, lastTs},
				Block:   time.Second,
				Count:   60,
			}

			vals, err := gih.redisConn.XRead(args).Result()
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
			// delay or 16 ms before new attempty
			time.Sleep(time.Millisecond * 16)
		}

		if errStr := stream.Send(vf); errStr != nil {
			g.Log.Error("send error", errStr)
		}
	}
}
