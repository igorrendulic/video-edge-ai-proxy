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
import sys

# create in-memory buffer gRPC request
def gen_buffered_image_request(device_name, timestamp_from, timestamp_to):
    """ Create an object to request a video frame """


    req = video_streaming_pb2.VideoFrameBufferedRequest()
    req.device_id = device_name
    req.timestamp_from = timestamp_from
    req.timestamp_to = timestamp_to
    return req



if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Chrysalis Edge buffered images example')
    parser.add_argument("--device", type=str, default=None, required=True)
    args = parser.parse_args()
    device_id = args.device

    # processing everything in the in-memory buffer

    # grpc connection to video-edge-ai-proxy
    options = [('grpc.max_receive_message_length', 50 * 1024 * 1024)]
    channel = grpc.insecure_channel('127.0.0.1:50001', options=options)
    stub = video_streaming_pb2_grpc.ImageStub(channel)

    # first get the system time (not necessary but sure to be more precise on different systems)
    sysTime = stub.SystemTime(video_streaming_pb2.SystemTimeRequest())
    timestampTo = sysTime.current_time
    timestampFrom = 0 # beginning of the in-memory queue

    num_images = 0
    start_time = int(time.time() * 1000)
    images_found = False

    for frame in stub.VideoBufferedImage(gen_buffered_image_request(device_name=device_id,timestamp_from=timestampFrom, timestamp_to=timestampTo)):
        # read raw frame data and convert to numpy array
        img_bytes = frame.data 
        re_img = np.frombuffer(img_bytes, dtype=np.uint8)
        
        # checking if any results found
        images_found = True

        # reshape image back into original dimensions
        if len(frame.shape.dim) > 0:
            reshape = tuple([int(dim.size) for dim in frame.shape.dim])
            re_img = np.reshape(re_img, reshape)

            # # display and count number of images retrieved from buffer
            num_images += 1
            cv2.imshow('box', re_img)
            
            if cv2.waitKey(1) & 0xFF == ord('q'):
                break
    
    if images_found: # don't destroy window if nothing ever displayed
        end_time = int(time.time() * 1000)

        print("Total execution time:", str(end_time - start_time), ": ", (end_time - start_time), "ms", "[Start:End] -> [", str(timestampFrom), ":", str(timestampTo), "]", "#images: ", num_images)
    else:
        print("no results found between ", timestampFrom, ",", timestampTo)
    pass