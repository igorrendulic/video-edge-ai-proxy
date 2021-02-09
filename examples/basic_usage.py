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


def send_list_stream_request(stub):
    """ Create a list of streams request object """


    stream_request = video_streaming_pb2.ListStreamRequest()   
    responses = stub.ListStreams(stream_request)
    for stream_resp in responses:
        yield stream_resp


def gen_image_request(device_name, keyframe_only):
    """ Create an object to request a video frame """


    req = video_streaming_pb2.VideoFrameRequest()
    req.device_id = device_name
    req.key_frame_only = keyframe_only
    return req


if __name__ == "__main__":
    # Initialize parser 
    parser = argparse.ArgumentParser(description='Chrysalis Edge Proxy Basic Example')
    parser.add_argument("--list", action='store_true')
    parser.add_argument("--device", type=str, default=None, required=False)

    args = parser.parse_args()
    
    # grpc connection to video-edge-ai-proxy
    options = [('grpc.max_receive_message_length', 50 * 1024 * 1024)]
    channel = grpc.insecure_channel('127.0.0.1:50001', options=options)
    stub = video_streaming_pb2_grpc.ImageStub(channel)

    if args.list:
        list_streams = send_list_stream_request(stub)
        for stream in list_streams:
            print(stream)
    
    if args.device:
        frame = stub.VideoLatestImage(gen_image_request(device_name=args.device,keyframe_only=False))
        print("is keyframe: ", frame.is_keyframe)
        print("frame type: ", frame.frame_type)
        print("frame shape: ", frame.shape)
