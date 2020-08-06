import time
import grpc
import video_streaming_pb2
import video_streaming_pb2_grpc
import threading
import numpy as np
import cv2
import os
from grpc_exponential_backoff import RetryOnRpcErrorClientInterceptor, ExponentialBackoff
from argparse import ArgumentParser
import sys

# required on Linux
os.environ['DISPLAY'] = ":0"

def gen_request(device_name, keyonly=False):
    req = video_streaming_pb2.VideoFrameRequest()
    req.device_id = device_name
    req.key_frame_only = keyonly
    yield req

def send_image_request(stub, name, keyframe_only=False):
    
    responses = stub.VideoLatestImage(gen_request(name, keyframe_only))
    for response in responses:
        img_bytes = response.data
        re_img = np.frombuffer(img_bytes, dtype=np.uint8)
        # width = response.width
        # height = response.height

        sh = response.shape
        tuple_list = []
        if len(sh.dim) > 0:
            for dim in sh.dim:
                tuple_list.append(int(dim.size))
            reshape = tuple(tuple_list)
            # Convert back the data to original image shape.
            re_img = np.reshape(re_img, reshape)
            
            print(response.packet, response.keyframe)

            cv2.imshow("RTSP Live View", re_img)
            
            if cv2.waitKey(1) & 0xFF == ord('q'):
                break

def send_list_stream_request(stub):
    stream_request = video_streaming_pb2.ListStreamRequest()   
    responses = stub.ListStreams(stream_request)
    for stream_resp in responses:
        yield stream_resp

def start_proxy(stub, device_name):
    start_proxy_request = video_streaming_pb2.StartProxyRequest()
    start_proxy_request.device_id = device_name
    try:
        resp = stub.StartProxy(start_proxy_request)
        print(resp)
    except grpc.RpcError as rpc_error_call:
        print("start proxy failed with", rpc_error_call.code(), rpc_error_call.details())

def stop_proxy(stub, device_name):
    stop_proxy_request = video_streaming_pb2.StopProxyRequest()
    stop_proxy_request.device_id = device_name
    try:
        resp = stub.StopProxy(stop_proxy_request)
        print(resp)
    except grpc.RpcError as rpc_error_call:
        print("stop proxy failed with", rpc_error_call.code(), rpc_error_call.details())

def run(stub, device_name, keyframe_only):
        while True:
            send_image_request(stub, device_name, keyframe_only)
            time.sleep(0.1)

if __name__ == "__main__":
    parser = ArgumentParser(description="Example using Chrysalis Edge Proxy")
    parser.add_argument("--keyframe", type=bool, default=False, required=False)
    parser.add_argument("--device", type=str, default=None, required=False)
    parser.add_argument("--list", type=bool, default=False, required=False)
    parser.add_argument("--startproxy", type=bool, default=False, required=False)
    parser.add_argument("--stopproxy", type=bool, default=False, required=False)

    args = results = parser.parse_args()

    device_name = args.device
    keyframes_only = args.keyframe

    list_device = args.list

    if not device_name and not list_device:
        sys.exit("required either device name or list parameter Type: python_example.py -h to see argument options")

    interceptors = (
        RetryOnRpcErrorClientInterceptor(
        max_attempts=4,
        sleeping_policy=ExponentialBackoff(init_backoff_ms=100, max_backoff_ms=1600, multiplier=2),
        status_for_retry=(grpc.StatusCode.UNAVAILABLE,),
        ),
    )

    with grpc.intercept_channel(grpc.insecure_channel("127.0.0.1:50001"), *interceptors) as channel:
        stub = video_streaming_pb2_grpc.ImageStub(channel)

        if list_device:
            print("---------------------DEVICES---------------------")
            devices = send_list_stream_request(stub)
            for dev in devices:
                print(dev)
        elif args.startproxy:
            start_proxy(stub, device_name)
        elif args.stopproxy:
            stop_proxy(stub, device_name)
        else:
            print("RUNNING DEVICE")
            print("Name: ", device_name)
            print("Keyframes only: ", keyframes_only)
            print("--------------")
            run(stub, device_name, keyframes_only)
