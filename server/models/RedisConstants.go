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

package models

// Constants for Redis stored keys, used to communicate between operating containers and use APIs
const (
	// RedisLastAccessPrefix - latest access time in milliseconds for particual deviceID/Name
	RedisLastAccessPrefix = "last_access_time_"
	// RedisIsKeyFrameOnlyPrefix - setting if only keyframes required for decoding for specific deviceID/Name
	RedisIsKeyFrameOnlyPrefix = "is_key_frame_only_"

	// RedisLastAccessPrefix subkeys for the HSET map
	RedisLastAccessQueryTimeKey = "last_query" // last request query time
	RedisProxyRTMPKey           = "proxy_rtmp" // if the RTMP should be proxied to the Chrysalis cloud
	RedisProxyStoreKey          = "store"      // if Chrysalis cloud should store the stream to permanent storage
)
