# Copyright 2020 Wearless Tech Inc All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

query_timestamp = None
RedisLastAccessPrefix = "last_access_time_"
RedisIsKeyFrameOnlyPrefix = "is_key_frame_only_"

# in memory constants
RedisInMemoryBufferChannel = "memory_buffer_channel"
RedisInMemoryQueuePrefix = "in_memory_queue_" # stored compressed video stream packet by packet
RedisInMemoryDecodedImagesPrefix = "memory_decoded_" # decoded video stream from in_memory_queue
RedisInMemoryIFrameListPrefix = "memory_iframe_list_" # helper list of all iframes for finding the closest i-frame faster

RedisCodecVideoInfo = "codec_video_info"

class ArchivePacketGroup():

    def __init__(self, packet_group, start_timestamp):
        self.packet_group = packet_group
        self.start_timestamp = start_timestamp

    def addPacket(self, packet=None):
        self.packet_group.append(packet)

    def setPacketGroup(self, packet_group=[]):
        self.packet_group = packet_group

    def setStartTimestamp(self, timestamp=None):
        self.start_timestamp = timestamp


class ChrysException(Exception):
    pass