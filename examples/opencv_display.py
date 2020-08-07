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
import video_streaming_pb2_grpc, video_streaming_pb2
import argparse
import cv2
import numpy as np

def gen_image_request(device_name, keyframe_only):
    """ Create an object to request a video frame """


    req = video_streaming_pb2.VideoFrameRequest()
    req.device_id = device_name
    req.key_frame_only = keyframe_only
    yield req

if __name__ == "__main__":
    
    parser = argparse.ArgumentParser(description='Chrysalis Edge Proxy Basic Example')
    parser.add_argument("--device", type=str, default=None, required=True)
    parser.add_argument("--keyframe", action='store_true')
    args = parser.parse_args()

    # grpc connection to video-edge-ai-proxy
    channel = grpc.insecure_channel('127.0.0.1:50001')
    stub = video_streaming_pb2_grpc.ImageStub(channel)
    
    while True:
        for frame in stub.VideoLatestImage(gen_image_request(device_name=args.device,keyframe_only=args.keyframe)):
            # read raw frame data and convert to numpy array
            img_bytes = frame.data 
            re_img = np.frombuffer(img_bytes, dtype=np.uint8)

            # reshape image back into original dimensions
            if len(frame.shape.dim) > 0:
                reshape = tuple([int(dim.size) for dim in frame.shape.dim])
                re_img = np.reshape(re_img, reshape)

                # add camera name
                cv2.putText(re_img, args.device, (20,20), cv2.FONT_HERSHEY_COMPLEX_SMALL, fontScale=1, color=(0, 0, 0), thickness=1)
                # display with opencv imshow
                cv2.imshow("RTSP Live View", re_img)
                
                if cv2.waitKey(1) & 0xFF == ord('q'):
                    break