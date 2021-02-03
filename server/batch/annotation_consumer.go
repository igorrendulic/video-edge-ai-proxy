package batch

import (
	"time"

	"github.com/adjust/rmq/v2"
	"github.com/chryscloud/go-microkit-plugins/models/ai"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/go-resty/resty/v2"
	"github.com/golang/protobuf/proto"
)

type AnnotationConsumer struct {
	settingsService *services.SettingsManager
	restClient      *resty.Client
	msgQueue        rmq.Queue
}

func NewAnnotationConsumer(tag int, settingsService *services.SettingsManager, msgQueue rmq.Queue) *AnnotationConsumer {
	restClient := resty.New().SetRetryCount(3)

	ac := &AnnotationConsumer{
		settingsService: settingsService,
		restClient:      restClient,
		msgQueue:        msgQueue,
	}

	// check every 5 seconds if any rejected annotations
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				ac.failedAnnotationsTryRedo(<-ticker.C)
			}
		}
	}()

	return ac
}

// checks if any messages need to be reinstated that have failed before (in case of internet outage on the edge)
func (ac *AnnotationConsumer) failedAnnotationsTryRedo(tick time.Time) {
	cnt := ac.msgQueue.ReturnAllRejected()
	if cnt > 0 {
		g.Log.Info("re-queued ", cnt, " of previously rejected annotatons")
	}
}

func (ac *AnnotationConsumer) Consume(batch rmq.Deliveries) {

	if g.Conf.Annotation.Endpoint == "" {
		g.Log.Error("expected annotation endpoint url. Check if you have /data/chrysalis/conf.yaml file")
		return
	}
	apiKey, apiSecret, err := ac.settingsService.GetCurrentEdgeKeyAndSecret()
	if err != nil {
		g.Log.Error("failed to retrieve edge api key and edge api secret", err)
		return
	}

	var aiAnnotations []*ai.Annotation

	for _, b := range batch {
		payload := []byte(b.Payload())
		var req pb.AnnotateRequest
		err := proto.Unmarshal(payload, &req)
		if err != nil {
			g.Log.Error("failed to unmarshal request proto in annotation consumer", err)
			// drop event
			continue
		}
		aiAnnotation := ac.RequestToAnnotation(&req)
		aiAnnotations = append(aiAnnotations, &aiAnnotation)
	}

	sendPayload := ai.AnnotationList{
		Data: aiAnnotations,
	}

	_, apiErr := utils.CallAPIWithBody(ac.restClient, "POST", g.Conf.Annotation.Endpoint, sendPayload, apiKey, apiSecret)
	if apiErr != nil {
		g.Log.Error("error calling Edge Annotation API", apiErr)
		batch.Reject()
	}

	batch.Ack()
}

// RequestToAnnotation (currently only REST supported on Chrysalis cloud. Later on GRPC just "push")
func (ac *AnnotationConsumer) RequestToAnnotation(req *pb.AnnotateRequest) ai.Annotation {
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

	return aiAnnotation
}
