package batch

import (
	"github.com/chryscloud/go-microkit-plugins/backpressure"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/go-resty/resty/v2"
)

type ChrysBatchWorker struct {
	chrysRestService *resty.Client
}

// NewChrysBatchWorker - init new batch worker
func NewChrysBatchWorker() *backpressure.PressureContext {
	restClient := resty.New()
	restClient.SetHostURL("https://monitordevapi.cocoon.health/api/v1/annotate")
	restClient.SetAuthToken("abc")
	bw := &ChrysBatchWorker{
		chrysRestService: restClient,
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
		// models := make([]*pb.AnnotateRequest, 0)
		g.Log.Info("processing annotations: ", len(annotations))
		for i := 0; i < len(annotations); i++ {
			// req := annotations[i].(*pb.AnnotateRequest)
			// g.Log.Info("annotation: ", req.DeviceName, req.Type, req.StartTimestamp)
		}
	}
	return nil
}
