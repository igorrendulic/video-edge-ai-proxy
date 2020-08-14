package batch

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/chryscloud/go-microkit-plugins/backpressure"
	microCrypto "github.com/chryscloud/go-microkit-plugins/crypto"
	"github.com/chryscloud/go-microkit-plugins/models/ai"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/go-resty/resty/v2"
)

type ChrysBatchWorker struct {
	restClient         *resty.Client
	settingsService    *services.SettingsManager
	annotationEndpoint string
}

// NewChrysBatchWorker - init new batch worker
func NewChrysBatchWorker(settingsService *services.SettingsManager) *backpressure.PressureContext {
	restClient := resty.New().SetRetryCount(3)

	bw := &ChrysBatchWorker{
		restClient:         restClient,
		settingsService:    settingsService,
		annotationEndpoint: g.Conf.Endpoints.AnnotationEndpoint,
	}
	bckPress, err := backpressure.NewBackpressureContext(bw,
		backpressure.BatchMaxSize(100),
		backpressure.BatchTimeMs(100),
		backpressure.Workers(1))

	if err != nil {
		g.Log.Error("failed to initialize backpressure", err)
		panic(err)
	}

	return bckPress
}

func (bw *ChrysBatchWorker) PutMulti(annotations []interface{}) error {
	if annotations != nil {

		if g.Conf.Endpoints.AnnotationEndpoint == "" {
			return errors.New("expected annotation endpoint url. Check if you have /data/chrysalis/conf.yaml file")
		}
		apiKey, apiSecret, err := bw.settingsService.GetCurrentEdgeKeyAndSecret()
		if err != nil {
			g.Log.Error("failed to retrieve edge api key and edge api secret", err)
			return err
		}
		g.Log.Info("API KEY: ", apiKey, apiSecret)

		var aiAnnotations []*ai.Annotation

		g.Log.Info("processing annotations: ", len(annotations))
		for i := 0; i < len(annotations); i++ {
			req := annotations[i].(*pb.AnnotateRequest)
			aiAnnotation := ai.Annotation{
				DeviceName:       req.DeviceName,
				Confidence:       req.Confidence,
				CustomMeta1:      req.CustomMeta_1,
				CustomMeta2:      req.CustomMeta_2,
				CustomMeta3:      req.CustomMeta_3,
				CustomMeta4:      req.CustomMeta_4,
				CustomMeta5:      req.CustomMeta_5,
				EndTimestamp:     req.EndTimestamp,
				StartTimestamp:   req.StartTimestamp,
				EventType:        req.Type,
				Height:           req.Height,
				Width:            req.Width,
				IsKeyframe:       req.IsKeyframe,
				MLModel:          req.MlModel,
				MLModelVersion:   req.MlModelVersion,
				ObjectID:         req.ObjectId,
				ObjectSignature:  req.ObjectSignature,
				ObjectTrackingID: req.ObjectTrackingId,
				ObjectType:       req.ObjectType,
				OffsetDuration:   req.OffsetDuration,
				OffsetFrameID:    req.OffsetFrameId,
				OffsetPAcketID:   req.OffsetPacketId,
				OffsetTimestamp:  req.OffsetTimestamp,
				RemoteStreamID:   req.RemoteStreamId,
				VideoType:        req.VideoType,
			}
			if req.Location != nil {
				aiAnnotation.Location = &ai.Location{
					Lat: req.Location.Lat,
					Lon: req.Location.Lon,
				}
			}
			if req.ObjectBoudingBox != nil {
				aiAnnotation.ObjectBoundingBox = &ai.BoundingBox{
					Height: req.ObjectBoudingBox.Height,
					Width:  req.ObjectBoudingBox.Width,
					Left:   req.ObjectBoudingBox.Left,
					Top:    req.ObjectBoudingBox.Top,
				}
			}
			if req.Mask != nil {
				var maskPolygon []*ai.Coordinate
				for _, m := range req.Mask {
					maskPolygon = append(maskPolygon, &ai.Coordinate{X: m.X, Y: m.Y, Z: m.Z})
				}
				aiAnnotation.ObjectMask = maskPolygon
			}
			aiAnnotations = append(aiAnnotations, &aiAnnotation)
		}

		sendPayload := ai.AnnotationList{
			Data: aiAnnotations,
		}
		payload, err := json.Marshal(sendPayload)
		if err != nil {
			g.Log.Error("invalid annotation json format", err)
			return err
		}

		h := md5.New()
		h.Write(payload)
		contentMD5 := hex.EncodeToString(h.Sum(nil))
		g.Log.Info("content md5: %v\n", contentMD5)
		current_ts := strconv.FormatInt(time.Now().Unix()*1000, 10)
		signPayload := current_ts + contentMD5
		g.Log.Info("payload: ", signPayload)
		mac := microCrypto.ComputeHmac(sha256.New, signPayload, apiSecret)
		g.Log.Info("MAC:", mac)

		resp, sndErr := bw.restClient.R().SetHeader("X-ChrysEdge-Auth", apiKey+":"+mac).
			SetHeader("X-Chrys-Date", current_ts).
			SetHeader("Content-MD5", contentMD5).SetBody(sendPayload).Post(bw.annotationEndpoint)

		if sndErr != nil {
			g.Log.Error("failed to send annotations to remote api", sndErr)
			return sndErr
		}
		if resp.StatusCode() >= 200 && resp.StatusCode() <= 300 {
			return nil
		} else {
			g.Log.Error("failed sending annotations: ", resp.StatusCode(), string(resp.Body()))
			return errors.New(fmt.Sprintf("failed to send annotations. Response from Chrysalis service: code -> %d", resp.StatusCode()))
		}
	}
	return nil
}
