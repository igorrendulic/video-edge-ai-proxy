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

package main

import (
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/chryscloud/go-microkit-plugins/backpressure"
	cfg "github.com/chryscloud/go-microkit-plugins/config"
	msrv "github.com/chryscloud/go-microkit-plugins/server"
	"github.com/chryscloud/video-edge-ai-proxy/batch"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/grpcapi"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	r "github.com/chryscloud/video-edge-ai-proxy/router"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	badger "github.com/dgraph-io/badger/v2"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

var (
	grpcServer    *grpc.Server
	grpcConn      net.Listener
	defaultDBPath = "/data/chrysalis"
)

// setup local badge datastore
func setupDB() (*badger.DB, error) {
	if _, err := os.Stat(defaultDBPath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		errDir := os.MkdirAll(defaultDBPath, os.ModePerm) //rw permission for the current user
		if errDir != nil {
			g.Log.Error("failed to create directiory for DB", defaultDBPath, errDir)
			return nil, errDir
		}
	}
	db, err := badger.Open(badger.DefaultOptions(defaultDBPath))
	if err != nil {
		g.Log.Error("faile to open database", err)
		return nil, err
	}
	return db, nil
}

func main() {
	// server wait to shutdown monitoring channels
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	// check if configuration file exists
	var conf g.Config
	if _, err := os.Stat(defaultDBPath + "/conf.yaml"); os.IsNotExist(err) {
		// config file does not exist
		conf = g.Config{
			YamlConfig: cfg.YamlConfig{
				Port: 8080,
				Mode: gin.ReleaseMode,
			},
		}
	} else {
		// custom config file exists
		err := cfg.NewYamlConfig(defaultDBPath+"/conf.yaml", &conf)
		conf.Port = 8080 // override port, if changed in config
		if err != nil {
			g.Log.Error(err, "conf.yaml failed to load")
			panic("Failed to load conf.yaml")
		}
	}
	g.Conf = conf

	signal.Notify(quit, os.Interrupt)
	defer signal.Stop(quit)

	db, err := setupDB()
	if err != nil {
		g.Log.Error("failed to init database", err)
		os.Exit(1)
	}
	defer db.Close()
	// Storage
	storage := services.NewStorage(db)

	// Services
	processService := services.NewProcessManager(storage)
	settingsService := services.NewSettingsManager(storage)
	annotationBatchService := batch.NewChrysBatchWorker(settingsService)
	defer annotationBatchService.Close()

	gin.SetMode(conf.Mode)

	router := msrv.NewAPIRouter(&conf.YamlConfig)
	router = r.ConfigAPI(router, processService, settingsService)

	// start server
	srv := msrv.Start(&conf.YamlConfig, router, g.Log)
	// wait for server shutdown
	go msrv.Shutdown(srv, g.Log, quit, done)

	go startGrpcServer(processService, settingsService, annotationBatchService)
	go shutdownGrpc(quit, done)

	g.Log.Info("Server is ready to handle requests at", conf.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		g.Log.Error("Could not listen on %s: %v\n", conf.Port, err)
	}

	<-done

	grpcConn.Close()
	g.Log.Info("exit")
}

func startGrpcServer(processService *services.ProcessManager, settingsService *services.SettingsManager, batchContext *backpressure.PressureContext) error {
	conn, err := net.Listen("tcp", "0.0.0.0:50001") // TODO: take from conf.yaml file
	if err != nil {
		g.Log.Error("Failed to open grpc connection", err)
		return err
	}
	grpcConn = conn
	grpcServer = grpc.NewServer()

	pb.RegisterImageServer(grpcServer, grpcapi.NewGrpcImageHandler(processService, settingsService, batchContext))
	g.Log.Info("Grpc Servier is ready to handle requests at 50001")
	return grpcServer.Serve(grpcConn)
}

func shutdownGrpc(quit <-chan os.Signal, done chan<- bool) {
	<-quit
	if grpcServer != nil {
		g.Log.Info("stopping grpc server...")
		grpcServer.GracefulStop()
	}
	close(done)
	g.Log.Info("stopping grpc server...")
}
