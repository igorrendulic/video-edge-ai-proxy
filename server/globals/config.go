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

package globals

import (
	cfg "github.com/chryscloud/go-microkit-plugins/config"
	mclog "github.com/chryscloud/go-microkit-plugins/log"
)

// Conf global config
var Conf Config

// Log global wide logging
var Log mclog.Logger

type Config struct {
	cfg.YamlConfig `yaml:",inline"`
	GrpcPort       string               `yaml:"grpc_port"`
	Redis          *RedisSubconfig      `yaml:"redis"`
	Annotation     *AnnotationSubconfig `yaml:"annotation"`
	API            *ApiSubconfig        `yaml:"api"`
	Buffer         *BufferSubconfig     `yaml:"buffer"`
}

// RedisSubconfig connnection settings
type RedisSubconfig struct {
	Connection string `yaml:"connection"`
	Database   int    `yaml:"database"`
	Password   string `yaml:"password"`
}

// AnnotationSubconfig - annotation consumer rates
type AnnotationSubconfig struct {
	Endpoint       string `yaml:"endpoint"`         // chryscloud annotation endpoint
	UnackedLimit   int    `yaml:"unacked_limit"`    // maximum number of unacknowledged annotations
	PollDurationMs int    `yaml:"poll_duration_ms"` // time to wait until new poll of annotations (miliseconds)
	MaxBatchSize   int    `yaml:"max_batch_size"`   // maximum number of events processed in one batch
}

// VideoApiSubconfig - video api specifics
type ApiSubconfig struct {
	Endpoint string `yaml:"endpoint"` // video storage on/off endpoint
}

// Buffer - in memory and on disk buffering
type BufferSubconfig struct {
	InMemory               int    `yaml:"in_memory"`                // number of decoded frames to store in memory per camera
	InMemoryScale          string `yaml:"in_memory_scale"`          // scale in-memory video to desired size (e.g.: default = "-1:-1" , "400:-1", "300x200", "iw/2:ih/2")
	OnDisk                 bool   `yaml:"on_disk"`                  // store key-frame segmented mp4 files to disk
	OnDiskCleanupOlderThan string `yaml:"on_disk_clean_older_than"` // clean up mp4 segments after X time
	OnDiskFolder           string `yaml:"on_disk_folder"`           // location to store mp4 segments
	OnDiskSchedule         string `yaml:"on_disk_schedule"`         // schedule cleanup every X duration
}

func init() {
	l, err := mclog.NewZapLogger("info")
	if err != nil {
		panic("failed to initalize logging")
	}
	Log = l
}
