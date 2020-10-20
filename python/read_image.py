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

import threading, queue
from proto import video_streaming_pb2
import numpy as np
import time
import redis
from global_vars import query_timestamp, RedisLastAccessPrefix, RedisIsKeyFrameOnlyPrefix

class ReadImage(threading.Thread):

    def __init__(self, packet_queue, device_id, memory_buffer, redis_conn, is_decode_packets_event, lock_condition):
        threading.Thread.__init__(self)
        self._packet_queue = packet_queue
        self.device_id = device_id
        self._memory_buffer = memory_buffer
        self.redis_conn = redis_conn
        self.is_decode_packets_event = is_decode_packets_event
        self.lock_condition = lock_condition
        self.last_query_timestamp = 0
        self.packet_group = []

    # checks if only keyframes requested
    def check_decode_only_keyframes(self):
        global RedisIsKeyFrameOnlyPrefix
        decode_only_keyframes = False
        decodeOnlyKeyFramesKey = RedisIsKeyFrameOnlyPrefix + self.device_id
        only_keyframes = self.redis_conn.get(decodeOnlyKeyFramesKey)
        if only_keyframes is not None:
            okeys = only_keyframes.decode('utf-8')
            if okeys.lower() == "true":
                decode_only_keyframes = True
        return decode_only_keyframes

    def run(self):
        global query_timestamp

        packet_count = 0
        keyframes_count = 0

        query_timestamp = self.last_query_timestamp

        while True:
            with self.lock_condition:
                self.lock_condition.wait()

                if not self._packet_queue.empty() and self.is_decode_packets_event.is_set():
                    try:
                        packet = self._packet_queue.get()

                        decode_only_keyframes = self.check_decode_only_keyframes()

                        if packet.is_keyframe:
                            self.packet_group = []
                            packet_count = 0
                            keyframes_count = keyframes_count + 1
                        
                        self.packet_group.append(packet)

                        should_decode = True
                        # if only keyframes, then decode only when len of packet_group == 1
                        if decode_only_keyframes:
                            should_decode = False

                        if len(self.packet_group) == 1 or should_decode: # by default decode every keyframe
                            for index, p in enumerate(self.packet_group):

                                # skip already decoded packets in this packet group
                                if index < packet_count:
                                    continue

                                for frame in p.decode() or ():
                                    
                                    timestamp = int(round(time.time() * 1000))
                                    if frame.time is not None:
                                        timestamp = int(frame.time * frame.time_base.denominator)

                                    # add numpy array byte to redis stream
                                    img = frame.to_ndarray(format='bgr24')
                                    shape = img.shape

                                    img_bytes = np.ndarray.tobytes(img)

                                    vf = video_streaming_pb2.VideoFrame()
                                    vf.data = img_bytes
                                    vf.width = frame.width
                                    vf.height = frame.height
                                    vf.timestamp = timestamp
                                    vf.frame_type = frame.pict_type.name
                                    vf.pts = frame.pts
                                    vf.dts = frame.dts
                                    vf.packet = packet_count
                                    vf.keyframe = keyframes_count
                                    vf.time_base = float(frame.time_base)
                                    vf.is_keyframe = packet.is_keyframe
                                    vf.is_corrupt = packet.is_corrupt

                                    for (i,dim) in enumerate(shape):
                                        newDim = video_streaming_pb2.ShapeProto.Dim()
                                        newDim.size = dim
                                        newDim.name = str(i)
                                        vf.shape.dim.append(newDim)

                                    vfData = vf.SerializeToString()

                                    self.redis_conn.xadd(self.device_id, {'data': vfData}, maxlen=self._memory_buffer)

                                    if decode_only_keyframes:
                                        break

                                    self.last_query_timestamp = query_timestamp

                                packet_count = packet_count + 1

                    except Exception as e:
                        print("failed to deode packet", e)
                    finally:
                        self._packet_queue.task_done()