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
import time


def storage(stub, device_name, onoff=False):
    """ Enabling or disabling storage on live RTMP stream """


    storage_request = video_streaming_pb2.StorageRequest()
    storage_request.device_id = device_name
    storage_request.start = onoff
    try:
        resp = stub.Storage(storage_request)
        print(resp)
    except grpc.RpcError as rpc_error_call:
        print("start proxy failed with", rpc_error_call.code(), rpc_error_call.details())

def str2bool(v):
    if isinstance(v, bool):
       return v
    if v.lower() in ('yes', 'true', 't', 'y', '1'):
        return True
    elif v.lower() in ('no', 'false', 'f', 'n', '0'):
        return False
    else:
        raise argparse.ArgumentTypeError('Boolean value expected.')

if __name__ == "__main__":
    # Initialize parser 
    parser = argparse.ArgumentParser(description='Chrysalis Edge Proxy Storage Example')
    parser.add_argument("--device", type=str, default=None, required=True)
    parser.add_argument("--on", type=str2bool, default=None, required=True)

    args = parser.parse_args()
    
    # grpc connection to video-edge-ai-proxy
    channel = grpc.insecure_channel('127.0.0.1:50001')
    stub = video_streaming_pb2_grpc.ImageStub(channel)

    # send annotation
    storage(stub, args.device, args.on)
