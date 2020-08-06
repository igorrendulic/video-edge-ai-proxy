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
}

func init() {
	l, err := mclog.NewZapLogger("info")
	if err != nil {
		panic("failed to initalize logging")
	}
	Log = l
}
