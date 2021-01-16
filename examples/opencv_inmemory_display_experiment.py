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


import grpc
from numpy.lib.function_base import append
import video_streaming_pb2_grpc, video_streaming_pb2
import argparse
import cv2
import numpy as np
import time
import os
import threading

class SegmentThread(threading.Thread):

    def __init__(self, query_from, query_to, device, segment_number, results):
        threading.Thread.__init__(self)
        self.__query_from = query_from
        self.__query_to = query_to
        self.__device_id = device
        self.__segment_number = segment_number
        self.__results = results

    def gen_buffered_image_request(device_name, timestamp_from, timestamp_to):
        """ Create an object to request a video frame """


        req = video_streaming_pb2.VideoFrameBufferedRequest()
        req.device_id = device_name
        req.timestamp_from = timestamp_from
        req.timestamp_to = timestamp_to
        return req

    def run(self):
        duration = self.__query_from - self.__query_to

        num_images = 0
        start_time = int(time.time() * 1000)
        images_found = False

        img_array = []

        for frame in stub.VideoBufferedImage(gen_buffered_image_request(device_name=self.__device_id,timestamp_from=self.__query_from, timestamp_to=self.__query_to)):
            # read raw frame data and convert to numpy array
            img_bytes = frame.data 
            re_img = np.frombuffer(img_bytes, dtype=np.uint8)
            
            # checking if any results found
            images_found = True

            # reshape image back into original dimensions
            if len(frame.shape.dim) > 0:
                reshape = tuple([int(dim.size) for dim in frame.shape.dim])
                re_img = np.reshape(re_img, reshape)
                img_array.append(re_img)
                num_images += 1
        
        if images_found: # don't destroy window if nothing ever displayed
            end_time = int(time.time() * 1000)

            print("Total execution time for segment ", str(self.__segment_number), ": ", (end_time - start_time), "ms", "[Start:End] -> [", str(start_query), ":", str(end_query), "]", "#images: ", num_images, "segment duration: ", duration)
            # cv2.destroyWindow('box')
            self.__results[self.__segment_number] = img_array
        else:
            print("no results found between ", timestampFrom, ",", timestampTo)
        pass

def gen_buffered_image_request(device_name, timestamp_from, timestamp_to):
    """ Create an object to request a video frame """


    req = video_streaming_pb2.VideoFrameBufferedRequest()
    req.device_id = device_name
    req.timestamp_from = timestamp_from
    req.timestamp_to = timestamp_to
    return req

def gen_buffer_probe_request(device_name, from_ts, to_ts):
    """ Create GRPC request to get in memory probe info """

    req = video_streaming_pb2.VideoBufferProbeRequest()
    req.device_id = device_name
    req.from_timestamp = from_ts
    req.to_timestamp = to_ts

    return req

if __name__ == "__main__":
    
    parser = argparse.ArgumentParser(description='Chrysalis Edge buffered images example')
    parser.add_argument("--device", type=str, default=None, required=True)
    args = parser.parse_args()
    device_id = args.device

    # timestampTo = int(time.time() * 1000) - (1000 * 60 * 2) # 4 minutes in the past
    # timestampFrom = timestampTo - (1000 * 60 * 1) # 5 minutes in the past
    timestampTo = int(time.time() * 1000)
    timestampFrom = timestampTo - (1000 * 50) # 6 seconds in the past

    # grpc connection to video-edge-ai-proxy
    options = [('grpc.max_receive_message_length', 50 * 1024 * 1024)]
    channel = grpc.insecure_channel('127.0.0.1:50001', options=options)
    stub = video_streaming_pb2_grpc.ImageStub(channel)

    # probing the in-memory video stream
    probe = stub.VideoBufferProbe(gen_buffer_probe_request(device_name=device_id, from_ts=timestampFrom, to_ts=timestampTo))
    print(probe)
    print("number of iframes: ", len(probe.iframes))
    iframes = probe.iframes

    total_start_time = int(time.time() * 1000)

    previous_image_query_time = 0

    images_found = False

    threads = []

    results = {}

    for i, iframe in enumerate(iframes):
        start_query = iframe.from_timestamp
        end_query = iframe.to_timestamp
        # duration = end_query - start_query

        # num_images = 0
        # start_time = int(time.time() * 1000)

        sThread = SegmentThread(start_query, end_query, device_id, i, results)
        sThread.daemon = True
        threads.append(sThread)

    for thread in threads:
        thread.start()

    for thread in threads:
        thread.join()

    sorted(results.keys())
    combined_list = []
    for k in results:
        print(k, len(results))
        combined_list.extend(results[k])

    print(type(combined_list), len(combined_list))
    end_time = int(time.time() * 1000)
    print("Total execution time: ", (end_time - total_start_time), "ms")

    for i,img in enumerate(combined_list):
        cv2.namedWindow('box', cv2.WINDOW_NORMAL)
        cv2.resizeWindow('box', 640,480)
        cv2.setWindowTitle('box', device_id) 
        cv2.imshow('box', img)
        
        if cv2.waitKey(1) & 0xFF == ord('q'):
            break