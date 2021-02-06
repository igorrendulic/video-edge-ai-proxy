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
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
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
	"github.com/rs/xid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageInput struct {
	Enable bool `json:"enable"`
}

type grpcImageHandler struct {
	redisConn       *redis.Client
	deviceMap       sync.Map
	processManager  *services.ProcessManager
	settingsManager *services.SettingsManager
	edgeKey         *string
	msgQueue        rmq.Queue
}

// NewGrpcImageHandler returns main GRPC API handler
func NewGrpcImageHandler(processManager *services.ProcessManager, settingsManager *services.SettingsManager, rdb *redis.Client) *grpcImageHandler {

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
	annotationConsumer := batch.NewAnnotationConsumer(0, settingsManager, msgQueue)
	msgQueue.StartConsuming(g.Conf.Annotation.UnackedLimit, time.Duration(g.Conf.Annotation.PollDurationMs)*time.Millisecond)
	msgQueue.AddBatchConsumerWithTimeout("annotationqueue", g.Conf.Annotation.MaxBatchSize, time.Duration(g.Conf.Annotation.PollDurationMs)*time.Millisecond, annotationConsumer)

	return &grpcImageHandler{
		redisConn:       rdb,
		deviceMap:       sync.Map{},
		processManager:  processManager,
		settingsManager: settingsManager,
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

// VideoLatestImage - bidirectional connection with client continously sending live video image
func (gih *grpcImageHandler) VideoLatestImage(stream pb.Image_VideoLatestImageServer) error {

	// clientDeadline := time.Now().Add(time.Duration(15) * time.Second)
	// streamContext, streamCancel := context.WithDeadline(stream.Context(), clientDeadline)
	// defer streamCancel()
	streamContext := context.Background()

	failedContinousRequests := 0

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
			g.Log.Warn("failed to retrieve gprc image request", err)
			if failedContinousRequests > 5 {
				return nil
			}
			failedContinousRequests++
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
		vf := &pb.VideoFrame{
			DeviceId: streamName,
		}

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
			g.Log.Error("grp live image send error", errStr)
		}
		failedContinousRequests = 0
	}
}

// VideoBufferProbe is a probing method for in-memory video stream
func (gih *grpcImageHandler) VideoProbe(ctx context.Context, req *pb.VideoProbeRequest) (*pb.VideoProbeResponse, error) {

	codecInfo := &pb.VideoCodec{}

	codecInfoCmd := gih.redisConn.Get(models.RedisCodecVideoInfo)
	codecInfoBytes, err := codecInfoCmd.Bytes()
	if err != nil {
		g.Log.Error("failed to get bytes for code info", err)
	} else {
		err := proto.Unmarshal(codecInfoBytes, codecInfo)
		if err != nil {
			g.Log.Error("failed to unmarshal codec info", err)
			return nil, status.Errorf(codes.Internal, "failed to unmarshal code info")
		}
	}

	resp := &pb.VideoProbeResponse{
		VideoCodec: codecInfo,
	}

	return resp, nil
}

// System time returns current systems time (taken from redis)
func (gih *grpcImageHandler) SystemTime(ctx context.Context, req *pb.SystemTimeRequest) (*pb.SystemTimeResponse, error) {
	sysTime := gih.redisConn.Time()
	t := sysTime.Val()

	resp := &pb.SystemTimeResponse{
		CurrentTime: t.UTC().Unix() * 1000,
	}

	return resp, nil
}

// VideoBufferedImage publishes a request for decoding to redis pub/sub and waits for decoded images to be taken out of Redis XSTREAM
func (gih *grpcImageHandler) VideoBufferedImage(req *pb.VideoFrameBufferedRequest, stream pb.Image_VideoBufferedImageServer) error {

	from := req.TimestampFrom
	to := req.TimestampTo
	deviceID := req.DeviceId

	pubsubMsg := &models.PubSubMessage{
		DeviceID:      deviceID,
		FromTimestamp: from,
		ToTimestamp:   to,
		RequestID:     xid.New().String(),
	}

	pubSubMsgBytes, err := json.Marshal(pubsubMsg)
	if err != nil {
		g.Log.Error("failed to marshal pubsub Msg", err)
	}
	pubSubBase64 := base64.StdEncoding.EncodeToString(pubSubMsgBytes)

	// method is waiting for the images to be decoded and put into a queue with the name streamName
	streamName := models.RedisInMemoryDecodedImagesPrefix + deviceID + pubsubMsg.RequestID

	// publish to redis request for decoding the queried in memory buffer
	gih.redisConn.Publish(models.RedisInMemoryBufferChannel, pubSubBase64)

	// read decoded images from the start
	lastRequestedTs := "0-0"

	// waiting up to 3 x 50ms = max to get an image from redis, otherwise considered that there is no image
	delayMs := 50

	// the time this whole process started (used to determine timeout)
	processStarted := time.Now().UTC().Unix() * 1000

	isReadingDone := false

	for {

		select {
		case <-stream.Context().Done():
			// memory cleanup

			gih.redisConn.Del(streamName)

			g.Log.Warn("Context done: ", stream.Context().Err())
			return stream.Context().Err()
		default:
		}

		if isReadingDone {
			break
		}

		currentTime := time.Now().UTC().Unix() * 1000

		if math.Abs(float64(currentTime-processStarted)) > (1000 * 15) {
			g.Log.Error("request timed out. Waiting for response 15s but nothing happened")
			break
		}

		args := &redis.XReadArgs{
			Streams: []string{streamName, lastRequestedTs},
			Block:   time.Microsecond * 50,
			Count:   10,
		}

		vals, err := gih.redisConn.XRead(args).Result()

		if err != nil {
			g.Log.Info("waiting for an image, retry #", err.Error())
			time.Sleep(time.Duration(delayMs))
			continue
		}
		processStarted = time.Now().Unix() * 1000
		for _, val := range vals {
			messages := val.Messages

			// loading VideoFrame from redis
			vf := &pb.VideoFrame{}

			idsToDelete := make([]string, 0)

			if len(messages) > 0 {
				// reading only latest image
				lastMessage := messages[len(messages)-1]
				id := lastMessage.ID
				lastRequestedTs = id

				idsToDelete = append(idsToDelete, id)

				object := lastMessage.Values
				if val, ok := object["data"]; ok {
					str := val.(string)
					b := []byte(str)
					err := proto.Unmarshal(b, vf)
					if err != nil {
						g.Log.Error("failed to unmarshall VideoFrame proto", err)
						break
					}
					if vf.Data == nil {
						isReadingDone = true
						g.Log.Info("succesfully retrieved in-memory buffer for query ", deviceID, " [ ", from, " : ", to, " ]")
					}
				}
				if !isReadingDone {
					if errStr := stream.Send(vf); errStr != nil {
						g.Log.Error("grp live image send error", errStr)
					}
				}
			}
			if len(idsToDelete) > 0 {
				delResp := gih.redisConn.XDel(streamName, idsToDelete...)
				_, delErr := delResp.Result()
				if delErr != nil {
					g.Log.Error("Failed to delete xstream read images", delErr)
				}
			}
		}
	}
	// delete the complete key (best effort)
	gih.redisConn.Del(streamName)

	return nil
}
