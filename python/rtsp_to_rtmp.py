import av
import time
import threading, queue
import os
import io
from av.filter import Filter, Graph
from av.codec import CodecContext
import redis
import numpy as np
from proto import video_streaming_pb2
import random
from argparse import ArgumentParser
import sys

query_timestamp = None
RedisLastAccessPrefix = "last_access_time_"
RedisIsKeyFrameOnlyPrefix = "is_key_frame_only_"

class ReadImage(threading.Thread):

    def __init__(self, packet_queue, device_id, redis_conn, is_decode_packets_event, lock_condition):
        threading.Thread.__init__(self)
        self._packet_queue = packet_queue
        self.device_id = device_id
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

                        should_decode = False
                        if query_timestamp is None:
                            should_decode = False
                        if query_timestamp > self.last_query_timestamp:
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

                                    self.redis_conn.xadd(self.device_id, {'data': vfData}, maxlen=60)

                                    if decode_only_keyframes:
                                        break

                                    self.last_query_timestamp = query_timestamp

                                packet_count = packet_count + 1

                    except Exception as e:
                        print("failed to deode packet", e)
                    finally:
                        self._packet_queue.task_done()

class RTSPtoRTMP(threading.Thread):

    def __init__(self, rtsp_endpoint, rtmp_endpoint, packet_queue, device_id, redis_conn, is_decode_packets_event, lock_condition):
        threading.Thread.__init__(self) 
        self._packet_queue = packet_queue
        self.rtsp_endpoint = rtsp_endpoint
        self.rtmp_endpoint = rtmp_endpoint
        self.redis_conn = redis_conn
        self.device_id = device_id
        self.is_decode_packets_event = is_decode_packets_event
        self.lock_condition = lock_condition
        self.query_timestamp = query_timestamp

    def link_nodes(self,*nodes):
        for c, n in zip(nodes, nodes[1:]):
            c.link_to(n)

    def run(self):
        global RedisLastAccessPrefix

        current_packet_group = []
        flush_current_packet_group = False

        should_mux = False

        while True:
            try:
                options = {'rtsp_transport': 'tcp', 'stimeout': '5000000', 'max_delay': '5000000', 'use_wallclock_as_timestamps':"1", "fflags":"+genpts", 'acodec':'aac'}
                self.in_container = av.open(self.rtsp_endpoint, options=options)
                self.in_video_stream = self.in_container.streams.video[0]
                self.in_audio_stream = None
                if len(self.in_container.streams.audio) > 0:
                    self.in_audio_stream = self.in_container.streams.audio[0]
            except Exception as ex:
                print("failed to connect to RTSP camera", ex)
                os._exit(1)
            
            keyframe_found = False
            global query_timestamp

            if self.rtmp_endpoint is not None:
                output = av.open(self.rtmp_endpoint, format="flv", mode='w')
                output_video_stream = output.add_stream(template=self.in_video_stream)  

            output_audio_stream = None
            if self.in_audio_stream is not None and self.rtmp_endpoint is not None:
                output_audio_stream = output.add_stream(template=self.in_audio_stream)


            for packet in self.in_container.demux(self.in_video_stream):

                if packet.dts is None:
                    continue
                
                if packet.is_keyframe:
                    # if we already found a keyframe previously, archive what we have
                    keyframe_found = True
                    current_packet_group = []
                
                if keyframe_found == False:
                    print("skipping, since not a keyframe")
                    continue
                
                # shouldn't be a problem for redis but maybe every 200ms to query for latest timestamp only
                settings_dict = self.redis_conn.hgetall(RedisLastAccessPrefix + device_id)

                if settings_dict is not None and len(settings_dict) > 0:
                    settings_dict = { y.decode('utf-8'): settings_dict.get(y).decode('utf-8') for y in settings_dict.keys() } 
                    ts = settings_dict['last_query']
                    should_mux_string = settings_dict['proxy_rtmp']
                    previous_should_mux = should_mux
                    if should_mux_string == "1":
                        should_mux = True
                    else:
                        should_mux = False
                    
                    # check if it's time for flushing of current_packet_group 
                    if should_mux != previous_should_mux and should_mux == True:
                        flush_current_packet_group = True
                    else:
                        flush_current_packet_group = False
                    
                    ts = int(ts)
                    ts_now = int(round(time.time() * 1000))
                    diff = ts_now - ts
                    # if no request in 10 seconds, stop
                    if diff < 10000:
                        try:
                            self.lock_condition.acquire()
                            query_timestamp = ts
                            self.lock_condition.notify_all()
                        finally:
                            self.lock_condition.release() 

                        self.is_decode_packets_event.set()

                if packet.is_keyframe:
                    self.is_decode_packets_event.clear()
                    self._packet_queue.queue.clear()
                
                
                self._packet_queue.put(packet)

                try:
                    if self.rtmp_endpoint is not None and should_mux:
                        # flush is necessary current_packet_group (start/stop RTMP stream)
                        if flush_current_packet_group:
                            for p in current_packet_group:
                                if p.stream.type == "video":
                                    p.stream = output_video_stream
                                    output.mux(p)
                                if p.stream.type == "audio":
                                    p.stream = output_audio_stream
                                    output.mux(p)

                        if packet.stream.type == "video":
                            packet.stream = output_video_stream
                            output.mux(packet)            
                        if packet.stream.type == "audio":
                            if output_audio_stream is not None:
                                packet.stream = output_audio_stream
                                output.mux(packet)
                except Exception as e:
                    print("failed muxing", e)

                current_packet_group.append(packet)
            
            time.sleep(1) # wait a second before trying to get more packets
            print("rtsp stopped streaming...waiting for camera to reappear")
        


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("--rtsp", type=str, default=None, required=True)
    parser.add_argument("--rtmp", type=str, default=None, required=False)
    parser.add_argument("--device_id", type=str, default=None, required=True)

    args = parser.parse_args()

    rtmp = args.rtmp
    rtsp = args.rtsp
    device_id = args.device_id

    decode_packet = threading.Event()
    lock_condition = threading.Condition()
    
    print("RTPS Endpoint: ",rtsp)
    print("RTMP Endpoint: ", rtmp)
    print("Device ID: ", device_id)

    redis_conn = None
    try:
        pool = redis.ConnectionPool(host="redis", port="6379")
        redis_conn = redis.Redis(connection_pool=pool)
    except Exception as ex:
        print("failed to connect to redis instance", ex)
        sys.exit("failed to connec to redis server")

    # Test redis connection
    ts = redis_conn.hgetall(RedisLastAccessPrefix + device_id)
    print("last query time: ", ts)

    get_images = False
    packet_queue = queue.Queue()

    th = RTSPtoRTMP(rtsp_endpoint=rtsp, 
                    rtmp_endpoint=rtmp, 
                    packet_queue=packet_queue, 
                    device_id=device_id, 
                    redis_conn=redis_conn, 
                    is_decode_packets_event=decode_packet, 
                    lock_condition=lock_condition)
    th.daemon = True
    th.start()

    ri = ReadImage(packet_queue=packet_queue, 
        device_id=device_id, 
        redis_conn=redis_conn, 
        is_decode_packets_event=decode_packet, 
        lock_condition=lock_condition)
    ri.daemon = True
    ri.start()
    ri.join()
    
    th.join()