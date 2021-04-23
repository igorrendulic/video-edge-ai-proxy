package api

import (
	"net/http"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/models"
	"github.com/chryscloud/video-edge-ai-proxy/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
)

type testApiHandler struct {
	rdb *redis.Client
}

func NewTestApiHandler(rdb *redis.Client) *testApiHandler {
	return &testApiHandler{
		rdb: rdb,
	}
}

// TestMqttDeviceStatus Testing mqtt device status
// @Security ApiKeyAuth
// @Summary Testing mqtt device status
// @Description Sends a mqtt message regarding device status
// @Tags MQTT
// @Param deviceId query string true "device name/id"
// @Success 200
// @Failure 417 {object} api.JSONError "test failed"
// @Accept json
// @Produce json
// @Router /testmqtt/api/v1/devicestatus [get]
func (th *testApiHandler) TestMqttDeviceStatus(c *gin.Context) {
	deviceId := c.Query("deviceId")

	processType := models.ProcessTypeRTSP
	err := utils.PublishToRedis(th.rdb, deviceId, models.MQTTProcessOperation(models.DeviceInternalTesting), processType, nil)
	if err != nil {
		g.Log.Error("test mqtt device status failed", err)
		AbortWithError(c, http.StatusExpectationFailed, err.Error())
		return
	}

	c.Status(http.StatusOK)
}
