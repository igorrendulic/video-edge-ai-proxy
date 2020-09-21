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
	"fmt"
	"os"
	"path/filepath"
	"time"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	"github.com/robfig/cron/v3"
)

func StartCronJobs(conf g.Config) []cron.EntryID {
	jobs := make([]cron.EntryID, 0)
	if conf.Buffer.OnDisk {
		err := startOnDiskCleanup(conf)
		if err != nil {
			panic(err.Error())
		}
	}
	return jobs
}

func startOnDiskCleanup(conf g.Config) error {
	c := cron.New(cron.WithLocation(time.UTC))

	// location, err := time.LoadLocation("EST")

	dur, err := time.ParseDuration(conf.Buffer.OnDiskCleanupOlderThan)
	if err != nil {
		g.Log.Error("failed to parse cron duration", err)
		return err
	}

	cId, err := c.AddFunc(g.Conf.Buffer.OnDiskSchedule, func() {
		folder := conf.Buffer.OnDiskFolder
		fpErr := filepath.Walk(folder, func(path string, info os.FileInfo, fErr error) error {
			if fErr != nil {
				g.Log.Error("failed to read file: ", fErr)
				return fErr
			}
			// calculating if file older than duration

			currentTime := time.Now().UTC()
			fileTime := info.ModTime().Add(dur)

			if fileTime.UTC().Before(currentTime) && !info.IsDir() && filepath.Ext(info.Name()) == ".mp4" {
				fmt.Printf("delete this file. it's too old: %v, %v %v, %v\n", path, dur, currentTime.UTC(), fileTime.UTC())
				remErr := os.Remove(path)
				if remErr != nil {
					g.Log.Error("failed to remove file: ", info.Name())
					return remErr
				}
			}
			return nil
		})
		if fpErr != nil {
			g.Log.Error("failed to scan buffer on_disk_files", fpErr)
		}
	})
	if err != nil {
		g.Log.Error("failed to start buffer on_disk_cleanup", err)
		return err
	}
	c.Start()
	g.Log.Info("started buffer on_disk_cleanup", cId)

	return nil
}
