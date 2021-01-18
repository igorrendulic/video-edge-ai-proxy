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


from multiprocessing import process
import grpc
import video_streaming_pb2_grpc, video_streaming_pb2
import argparse
import cv2
import numpy as np
import time
import multiprocessing

# request for vidoe from in memory buffer
def gen_buffered_image_request(device_name, timestamp_from, timestamp_to):
    """ Create an object to request a video frame """


    req = video_streaming_pb2.VideoFrameBufferedRequest()
    req.device_id = device_name
    req.timestamp_from = timestamp_from
    req.timestamp_to = timestamp_to
    return req

def grpc_stub(channel):
    # grpc connection to video-edge-ai-proxy a.k.a Chrysalis Edge Proxy
    return video_streaming_pb2_grpc.ImageStub(channel)

def grpc_channel(server_url):
    # in case of large images set limit to 50MB for reponses
    options = [('grpc.max_receive_message_length', 50 * 1024 * 1024)]
    channel = grpc.insecure_channel(server_url, options=options)
    return channel

# starting a video process
def video_process(queue, device,ts_from, ts_to, grpc_channel, process_number):

    stub = grpc_stub(grpc_channel)

    img_count = 0

    for frame in stub.VideoBufferedImage(gen_buffered_image_request(device_name=device,timestamp_from=ts_from, timestamp_to=ts_to)):
        # read raw frame data and convert to numpy array
        img_bytes = frame.data 
        re_img = np.frombuffer(img_bytes, dtype=np.uint8)
        # reshape image back into original dimensions
        if len(frame.shape.dim) > 0:
            reshape = tuple([int(dim.size) for dim in frame.shape.dim])
            re_img = np.reshape(re_img, reshape)

            img_count += 1

    #         queue.put({"img":re_img, "process": process_number})
    print("image count: ", img_count)
    
    # queue.put({"is_end":True, "process":process_number})

# debug function, checkin if dislay smooth and looks fastforwarded without any glitches
def display(num_processes):

    finished_processes = 0
    image_count = 0
    cam_images = [None] * num_processes

    while True:
        result = queue.get()

        process_num = result["process"]

        if "is_end" in result:
            print("finished processing ", process_num)
            finished_processes += 1

        if finished_processes >= num_processes:
            print("exit loop")
            break

        if "img" in result:
            image_count += 1
            img = result["img"]
            cam_images[process_num] = img

        # start displaying only when both left and right side are displaying
        if not any(v is None for v in cam_images):
            # concatinate horizontally
            tile = cv2.vconcat([cv2.hconcat(im_list_h) for im_list_h in [cam_images]])
            # display cameras
            cv2.imshow("test", tile)

            cv2.waitKey(1)

    print("image count: ", image_count)
        

if __name__ == "__main__":
    
    parser = argparse.ArgumentParser(description='Chrysalis Edge buffered images example')
    parser.add_argument("--device", type=str, default=None, required=True)
    args = parser.parse_args()
    device_id = args.device

    timestampTo = int(time.time() * 1000)
    timestampFrom = timestampTo - (1000 * 50) # 50 seconds in the past
    # deducting 3 seconds due to GOP (depends on camera settings). Idea: better to get a bit of video overlap then miss out a complete segmetn between I-farmes
    middleTimestampTo = int(timestampFrom + ((timestampTo - timestampFrom) / 2)) - (10 * 1000) 

    # grpc connection to video-edge-ai-proxy
    channel = grpc_channel("127.0.0.1:50001")
    stub = grpc_stub(channel)

    start_time = int(time.time() * 1000)

    # multiprocessing
    processes = []
    queue = multiprocessing.Queue()

    p1 = multiprocessing.Process(target=video_process, args=(queue, device_id, timestampFrom, middleTimestampTo, channel, 0, ))
    p2 = multiprocessing.Process(target=video_process, args=(queue, device_id, middleTimestampTo, timestampTo, channel, 1, ))
    processes.append(p1)
    processes.append(p2)

    for p in processes:
        p.start()

    # display(len(processes))

    for p in processes:
        p.join()

    print("Total execution time: ", int(time.time() * 1000) - start_time)
    



