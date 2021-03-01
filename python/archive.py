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

import av
import time
import threading, queue
import os
import datetime
from random import randint

class StoreMP4VideoChunks(threading.Thread):

    def __init__(self, queue=None, path=None, device_id=None, video_stream=None, audio_stream=None):
        threading.Thread.__init__(self) 
        self.in_video_stream = video_stream
        self.in_audio_stream = audio_stream
        self.path = os.path.join(path, '') + device_id
        self.device_id = device_id
        self.q = queue
        if not os.path.exists(self.path):
            os.makedirs(self.path)
    
    def run(self):
        while True:
            try:
                archive_group = self.q.get(timeout=5) # 5s timeout
                # print(archive_group.packet_group, archive_group.start_timestamp)
                self.saveToMp4(archive_group.packet_group, archive_group.start_timestamp)
            except queue.Empty:
                continue

            self.q.task_done()
        pass

    def saveToMp4(self, packet_store, start_timestamp):
        minimum_dts = -1
        maximum_dts = 0

        hasDuration = False
        segment_length = 0.0
        time_base = 0

        for _,p in enumerate(packet_store):
            sec = float(p.duration * p.stream.time_base) 
            if p.duration > 0:
                hasDuration = True
                segment_length += sec

            else:
                # calculation for some "older" cameras, that don't send duration in the packets
                if minimum_dts < 0:
                    minimum_dts = p.dts
            
                minimum_dts = min(minimum_dts, p.dts)
                maximum_dts = max(maximum_dts, p.dts)
                time_base = p.stream.time_base

        if not hasDuration:
            # calculate segment_length with time_base from a cam that has no duration info in stream 
            segment_length = (maximum_dts - minimum_dts) * time_base

        segment_length = int(segment_length * 1000) # convert to milliseconds

        output_file_name = self.path + "/" + str(start_timestamp) + "_" + str(segment_length) + ".mp4"
        output = av.open(output_file_name, format="mp4", mode='w')
        output_video_stream = output.add_stream(template=self.in_video_stream) 
        if self.in_audio_stream:
            print(self.in_audio_stream)
            # output_audio_stream = output.add_stream(template=self.in_audio_stream) 

        for _,p in enumerate(packet_store):
            p.dts -= minimum_dts
            p.pts -= minimum_dts
            # print(p.dts, p.pts)

        for _,p in enumerate(packet_store):
            # print ("PRE ", p, p.dts, p.pts, p.stream.type)
            if (p.stream.type == "video"):
                if p.dts is None:
                    continue

                p.stream = output_video_stream
                try:
                    output.mux(p)            
                except:
                    print("dts invalid probably")
                    continue
            # if (p.stream.type == "audio") and self.in_audio_stream:
            #     p.stream = output_audio_stream
            #     output.mux(p)
            # print ("POST ", p, p.dts, p.pts, p.stream.type)


        
        output.close()