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
	"container/list"
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"strconv"
	"strings"
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
	redisConn               *redis.Client
	deviceMap               sync.Map
	processManager          *services.ProcessManager
	settingsManager         *services.SettingsManager
	edgeKey                 *string
	msgQueue                rmq.Queue
	realtimeCache           sync.Map
	realtimeDeviceQueryTime sync.Map
}

// NewGrpcImageHandler returns main GRPC API handler
func NewGrpcImageHandler(processManager *services.ProcessManager, settingsManager *services.SettingsManager, rdb *redis.Client) *grpcImageHandler {

	conn := rmq.OpenConnectionWithRedisClient("annotationService", rdb)
	msgQueue := conn.OpenQueue("annotationqueue")

	// add batch listener (consumer) for annotatons
	annotationConsumer := batch.NewAnnotationConsumer(0, settingsManager, msgQueue)
	msgQueue.StartConsuming(g.Conf.Annotation.UnackedLimit, time.Duration(g.Conf.Annotation.PollDurationMs)*time.Millisecond)
	msgQueue.AddBatchConsumerWithTimeout("annotationqueue", g.Conf.Annotation.MaxBatchSize, time.Duration(g.Conf.Annotation.PollDurationMs)*time.Millisecond, annotationConsumer)

	return &grpcImageHandler{
		redisConn:               rdb,
		deviceMap:               sync.Map{},
		processManager:          processManager,
		settingsManager:         settingsManager,
		msgQueue:                msgQueue,
		realtimeCache:           sync.Map{},
		realtimeDeviceQueryTime: sync.Map{},
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
func (gih *grpcImageHandler) VideoLatestImage(ctx context.Context, request *pb.VideoFrameRequest) (*pb.VideoFrame, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	streamName := request.DeviceId

	// every 5 seconds report last query time
	currentTime := time.Now().UnixNano() / int64(time.Millisecond)

	lastQTime := int64(0)
	if lastDeviceQueryTime, ok := gih.realtimeDeviceQueryTime.Load(request.DeviceId); ok {
		lastQTime = lastDeviceQueryTime.(int64)
	}
	if currentTime-lastQTime > (5000) {
		// g.Log.Info("updating last query time for", request.DeviceId)
		// storeValuesStart := time.Now().UnixNano()

		isKeyFrameOnly := request.KeyFrameOnly

		decodeOnlyKeyFramesKey := models.RedisIsKeyFrameOnlyPrefix + streamName
		err := gih.redisConn.Set(decodeOnlyKeyFramesKey, strconv.FormatBool(isKeyFrameOnly), 0).Err()
		if err != nil {
			g.Log.Error("failed to set if is keyframe only", streamName, err)
			return nil, status.Errorf(codes.Internal, "failed to set preferences in redis")
		}

		valMap := make(map[string]interface{}, 0)
		valMap[models.RedisLastAccessQueryTimeKey] = currentTime

		rErr := gih.redisConn.HSet(models.RedisLastAccessPrefix+streamName, valMap).Err()
		if rErr != nil {
			g.Log.Error("failed to update on stopProxy redis", streamName, rErr)
			return nil, status.Errorf(codes.Internal, "can't access redis")
		}

		// storeValueEnd := time.Now().UnixNano()
		// g.Log.Info("Time to store query values [ms] ", (storeValueEnd-storeValuesStart)/1e+6)
		gih.realtimeDeviceQueryTime.Store(request.DeviceId, currentTime)
	}

	// // loading VideoFrame from redis
	vf := &pb.VideoFrame{}

	isDeviceFirstRun := false

	for i := 0; i < 3; i++ {

		var cache *list.List
		if cacheVal, ok := gih.realtimeCache.Load(request.DeviceId); ok {
			cache = cacheVal.(*list.List)
		} else {
			cache = list.New()
			gih.realtimeCache.Store(request.DeviceId, cache)
			isDeviceFirstRun = true
		}
		if isDeviceFirstRun {
			go func() {
				gih.cacheLiveVideo(request.DeviceId)
			}()

			time.Sleep(time.Millisecond * 200)
		}

		if cache.Len() > 0 {
			// g.Log.Info("Cache length: ", request.DeviceId, cache.Len())
			front := cache.Front()
			if front != nil {
				redisVal := front.Value.(redis.XMessage)
				vf = gih.unmarshalRedisImage(vf, request.DeviceId, redisVal)

				// if cache.Len() > 1 {
				cache.Remove(front)
				// }
				break
			}
		} else {
			time.Sleep(time.Millisecond * 16)
		}
	}

	return vf, nil
}

func (gih *grpcImageHandler) cacheLiveVideo(deviceId string) {

	for {

		lastTs := "0"
		if last_ts, ok := gih.deviceMap.Load(deviceId); ok {
			lastTs = last_ts.(string)
		}

		currentTime := time.Now().Unix() * 1000

		lastQTime := int64(0)
		if lastDeviceQueryTime, ok := gih.realtimeDeviceQueryTime.Load(deviceId); ok {
			lastQTime = lastDeviceQueryTime.(int64)
		}

		if lastTs != "0" && (currentTime-lastQTime) > (10*1000) { // if no query in the past 10 seconds stop caching
			g.Log.Info("stopping and cleaning cache for livestream ", deviceId)
			if cacheVal, ok := gih.realtimeCache.Load(deviceId); ok {
				fifoQueue := cacheVal.(*list.List)
				for j := 0; j < fifoQueue.Len()-10; j++ {
					fr := fifoQueue.Front()
					fifoQueue.Remove(fr)
				}
				gih.realtimeCache.Delete(deviceId)
				gih.deviceMap.Delete(deviceId)
				gih.realtimeDeviceQueryTime.Delete(deviceId)
			}
			break
		}

		args := &redis.XReadArgs{
			Streams: []string{deviceId, lastTs},
			Block:   time.Millisecond * 32,
			Count:   10,
		}

		vals, err := gih.redisConn.XRead(args).Result()
		if err != nil {
			// g.Log.Info("no image ready yet, returning nil#", deviceId, err.Error())
			continue
		}

		// add all redis.Xstream values to cache, except the first one (which is returned)
		if len(vals) > 0 {
			for _, val := range vals {
				if len(val.Messages) > 0 {

					var fifoQueue *list.List
					if cacheVal, ok := gih.realtimeCache.Load(deviceId); ok {
						fifoQueue = cacheVal.(*list.List)
					} else {
						fifoQueue = list.New()
					}

					// managing queue not to exceed 10 frames (frame dropping if client queries slower than frames are produced by the camera)
					for fifoQueue.Len() >= 10 {
						fr := fifoQueue.Front()
						fifoQueue.Remove(fr)
					}

					for i := 0; i < len(val.Messages); i++ {
						msg := val.Messages[i]
						// add to cache
						fifoQueue.PushBack(msg)

						if i == len(val.Messages)-1 {
							gih.deviceMap.Store(deviceId, msg.ID)
						}
					}
					gih.realtimeCache.Store(deviceId, fifoQueue)
				}
			}
		}
	}
}

func (gih *grpcImageHandler) unmarshalRedisImage(vf *pb.VideoFrame, deviceId string, msg redis.XMessage) *pb.VideoFrame {

	object := msg.Values
	if val, ok := object["data"]; ok {
		str := val.(string)
		b := []byte(str)
		err := proto.Unmarshal(b, vf)
		if err != nil {
			g.Log.Error("failed to unmarshall VideoFrame proto", err)
		}
		vf.DeviceId = deviceId
	}

	return vf
}

// VideoBufferProbe is a probing method for in-memory video stream
func (gih *grpcImageHandler) VideoProbe(ctx context.Context, req *pb.VideoProbeRequest) (*pb.VideoProbeResponse, error) {

	videoBuffer := &pb.VideoBuffer{}
	codecInfo := &pb.VideoCodec{}

	codecInfoCmd := gih.redisConn.Get(models.RedisCodecVideoInfo + req.DeviceId)
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

	lastFrame := gih.redisConn.XRevRangeN(models.RedisInMemoryQueue+req.DeviceId, "+", "-", 1)
	firstFrame := gih.redisConn.XRangeN(models.RedisInMemoryQueue+req.DeviceId, "-", "+", 1)

	// fps := 0
	length, err := gih.redisConn.XLen(models.RedisInMemoryQueue + req.DeviceId).Result()
	if err != nil {
		g.Log.Error("failed to approximate fps")
	}

	startTs := int64(0)
	endTs := int64(0)

	first, err := firstFrame.Result()
	last, err := lastFrame.Result()
	if err != nil {
		g.Log.Error("no in memory buffer for ", req.DeviceId)
	} else {
		startTs = gih.parseRedisTimestamp(first)
		endTs = gih.parseRedisTimestamp(last)
	}

	if startTs > 0 {
		videoBuffer.StartTime = startTs
		videoBuffer.EndTime = endTs
		videoBuffer.DurationSeconds = int64((endTs - startTs) / 1000)

		frameEveryMs := float64(videoBuffer.DurationSeconds) / float64(length)
		approxFps := float64(1) / frameEveryMs

		videoBuffer.ApproximateFps = int32(approxFps)
		videoBuffer.Frames = length
	}

	resp := &pb.VideoProbeResponse{
		VideoCodec: codecInfo,
	}
	if videoBuffer.StartTime > 0 {
		resp.Buffer = videoBuffer
	}

	return resp, nil
}

func (gih *grpcImageHandler) parseRedisTimestamp(msg []redis.XMessage) int64 {
	ts := int64(0)
	if len(msg) > 0 {
		msg := msg[0]
		splitted := strings.Split(msg.ID, "-")
		if len(splitted) == 2 {
			startTs, err := strconv.ParseInt(splitted[0], 10, 64)
			if err != nil {
				g.Log.Error("failed to parse probe start timestamp", err)
			} else {
				ts = startTs
			}
		}
	}
	return ts
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
						g.Log.Error("grpc buffered image send error", errStr)
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
