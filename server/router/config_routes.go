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

package router

import (
	api "github.com/chryscloud/video-edge-ai-proxy/api"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// ConfigAPI - configuring RESTapi services
func ConfigAPI(router *gin.Engine, processService *services.ProcessManager, settingsService *services.SettingsManager) *gin.Engine {

	// if g.Conf.CorsSubConfig.Enabled {
	router.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
	}))

	// APIs
	processAPI := api.NewRTSPProcessHandler(processService)
	settingsAPI := api.NewSettingsHandler(settingsService)

	api := router.Group("/api/v1")
	{
		api.POST("process", processAPI.Start)
		api.DELETE("process/:name", processAPI.Stop)
		api.GET("process/:name", processAPI.Info)
		api.GET("processlist", processAPI.List)
		api.GET("settings", settingsAPI.Get)
		api.POST("settings", settingsAPI.Overwrite)
	}

	return router
}
