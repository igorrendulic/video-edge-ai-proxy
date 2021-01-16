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
	"time"

	cfg "github.com/chryscloud/go-microkit-plugins/config"
	msrv "github.com/chryscloud/go-microkit-plugins/server"
	"github.com/chryscloud/video-edge-ai-proxy/globals"
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/chryscloud/video-edge-ai-proxy/grpcapi"
	pb "github.com/chryscloud/video-edge-ai-proxy/proto"
	r "github.com/chryscloud/video-edge-ai-proxy/router"
	"github.com/chryscloud/video-edge-ai-proxy/services"
	badger "github.com/dgraph-io/badger/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
	"google.golang.org/grpc"
)

var (
	grpcServer *grpc.Server
	grpcConn   net.Listener
	// defaultDBPath = "/data/chrysalis"
	defaultDBPath = "/home/igor/Downloads"
)

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
				Port: 8909,
				Mode: gin.ReleaseMode,
			},
		}
		conf.Annotation = &globals.AnnotationSubconfig{
			Endpoint:       "https://event.chryscloud.com/api/v1/annotate",
			MaxBatchSize:   299,
			PollDurationMs: 300,
			UnackedLimit:   1000,
		}
		conf.API = &globals.ApiSubconfig{
			Endpoint: "https://api.chryscloud.com",
		}
		conf.Redis = &globals.RedisSubconfig{
			Connection: "redis:6379",
			Database:   0,
			Password:   "",
		}
		conf.Buffer = &globals.BufferSubconfig{
			InMemory:               1,
			OnDisk:                 false,
			OnDiskCleanupOlderThan: "30s",
			OnDiskSchedule:         "@every 5m",
		}
	} else {
		// custom config file exists
		err := cfg.NewYamlConfig(defaultDBPath+"/conf.yaml", &conf)
		conf.Port = 8909 // override port, if changed in config
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

	rdb, rdbErr := setupRedis()
	if rdbErr != nil {
		g.Log.Error("Failed to init redis", rdbErr)
		os.Exit(1)
	}
	defer rdb.Close()

	// Storage
	storage := services.NewStorage(db)

	// Services
	settingsService := services.NewSettingsManager(storage)
	processService := services.NewProcessManager(storage, rdb)
	edgeService := services.NewEdgeService()

	gin.SetMode(conf.Mode)

	router := msrv.NewAPIRouter(&conf.YamlConfig)
	router = r.ConfigAPI(router, processService, settingsService)

	// start server
	srv := msrv.Start(&conf.YamlConfig, router, g.Log)
	// wait for server shutdown
	go msrv.Shutdown(srv, g.Log, quit, done)

	go startGrpcServer(processService, settingsService, edgeService, rdb)
	go shutdownGrpc(quit, done)

	g.Log.Info("Server is ready to handle requests at", conf.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		g.Log.Error("Could not listen on %s: %v\n", conf.Port, err)
	}

	<-done

	grpcConn.Close()
	g.Log.Info("exit")
}

func startGrpcServer(processService *services.ProcessManager, settingsService *services.SettingsManager, edgeService *services.EdgeService, rdb *redis.Client) error {
	conn, err := net.Listen("tcp", "0.0.0.0:50001") // TODO: take from conf.yaml file
	if err != nil {
		g.Log.Error("Failed to open grpc connection", err)
		return err
	}
	grpcConn = conn
	grpcServer = grpc.NewServer()

	pb.RegisterImageServer(grpcServer, grpcapi.NewGrpcImageHandler(processService, settingsService, edgeService, rdb))
	g.Log.Info("Grpc Server is ready to handle requests at 50001")
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

// setup redis datastore
func setupRedis() (*redis.Client, error) {
	var rdb *redis.Client
	for i := 0; i < 3; i++ {
		rdb = redis.NewClient(&redis.Options{
			Addr:        g.Conf.Redis.Connection,
			Password:    g.Conf.Redis.Password,
			DB:          g.Conf.Redis.Database,
			DialTimeout: time.Second * 15,
		})

		status := rdb.Ping()
		g.Log.Info("redis status: ", status)
		if status.Err() != nil {
			g.Log.Warn("waiting for redis to boot up", status.Err().Error)
			time.Sleep(3 * time.Second)
			continue
		}
		if i == 2 {
			return nil, status.Err()
		}
		break
	}
	return rdb, nil
}
