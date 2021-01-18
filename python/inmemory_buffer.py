import threading
from typing import MutableSequence
import av
import base64
import redis
import json
import sys
import io
import numpy as np
import time
from proto import video_streaming_pb2
import multiprocessing

# constants from global vars
from global_vars import RedisInMemoryBufferChannel,RedisInMemoryDecodedImagesPrefix, RedisInMemoryIFrameListPrefix,RedisCodecVideoInfo,RedisInMemoryQueuePrefix

def memoryCleanup(redis_conn, device_id):
    '''
    Cleanup redis memory
    '''
    redis_conn.delete(RedisInMemoryQueuePrefix+device_id) # the complete memory buffer of compressed stream
    redis_conn.delete(RedisInMemoryIFrameListPrefix+device_id) # all keys for stored i-frames
    redis_conn.delete(RedisInMemoryDecodedImagesPrefix+device_id) # all decoded in-memory buffer images

def setCodecInfo(redis_conn, in_av_container):
    '''
    Sets the current streams codec info at the same time clean out the in memory redis queues
    '''
    streams = in_av_container.streams
    if len(streams) > 0:
        for stream in streams:
            if stream.type == "video":

                codec_ctx = stream.codec_context
                vc = video_streaming_pb2.VideoCodec()
                vc.name = codec_ctx.name
                vc.long_name = codec_ctx.codec.long_name
                vc.width = codec_ctx.width
                vc.height = codec_ctx.height
                vc.pix_fmt = codec_ctx.pix_fmt
                vc.extradata = codec_ctx.extradata
                vc.extradata_size = codec_ctx.extradata_size

                vcData = vc.SerializeToString()
                redis_conn.set(RedisCodecVideoInfo, vcData)


def getCodecInfo(redis_conn):
    '''
    Reading the current video stream codec info from redis
    '''
    info = redis_conn.get(RedisCodecVideoInfo)
    if info is not None:
        vc = video_streaming_pb2.VideoCodec()
        vc.ParseFromString(info)
        return vc
    return None

def packetToInMemoryBuffer(redis_conn,memory_buffer_size, device_id,in_av_container, packet):
    if memory_buffer_size > 0:
        
        redisStreamName = RedisInMemoryQueuePrefix + device_id
        redisIFrameList = RedisInMemoryIFrameListPrefix + device_id

        for stream in in_av_container.streams:
            if stream.type == "video":
                codec_ctx = stream.codec_context
                video_height = codec_ctx.height
                video_width = codec_ctx.width
                is_keyframe = packet.is_keyframe
                packetBytes = packet.to_bytes()
                codec_name = codec_ctx.name
                pix_fmt = codec_ctx.pix_fmt

                vf = video_streaming_pb2.VideoFrame()
                vf.data = packetBytes
                vf.width = video_width
                vf.height = video_height
                vf.timestamp = int(packet.pts * float(packet.time_base))
                vf.pts = packet.pts
                vf.dts = packet.dts
                vf.keyframe = is_keyframe
                vf.time_base = float(packet.time_base)
                vf.is_keyframe = packet.is_keyframe
                vf.is_corrupt = packet.is_corrupt
                vf.codec_name = codec_name
                vf.pix_fmt = pix_fmt

                vfData = vf.SerializeToString()
                keyframe = 0
                if is_keyframe:
                    keyframe = 1
                    redis_conn.xadd(redisIFrameList, {'keyframe':keyframe}, maxlen=memory_buffer_size)

                redis_conn.xadd(redisStreamName, {'data': vfData, 'is_keyframe': keyframe}, maxlen=memory_buffer_size)


class InMemoryBuffer(threading.Thread):
    '''
    InMemoryBuffer stores packet by packet incoming video stream to redis queue
    '''
    def __init__(self, device_id, memory_scale, redis_conn):
        threading.Thread.__init__(self)

        self.__redis_conn = redis_conn
        self.__device_id = device_id
        self.__filter_scale = memory_scale


    def run(self):

        codec_info = getCodecInfo(self.__redis_conn)

        while codec_info is None:
            codec_info = getCodecInfo(self.__redis_conn)
            time.sleep(0.1)


        ps = self.__redis_conn.pubsub()
        ps.subscribe(RedisInMemoryBufferChannel)
        for psMsg in ps.listen():
            if "data" in psMsg:
                base64Msg = psMsg["data"]
                if isinstance(base64Msg, (bytes, bytearray)):
                    data = json.loads(base64.b64decode(base64Msg))

                    if "deviceId" in data:
                        deviceId = data["deviceId"]
                        fromTs = data["fromTimestamp"]
                        toTs = data["toTimestamp"]
                        requestID = data["requestId"]

                        p = multiprocessing.Process(target=self.query_results, args=(codec_info, requestID, deviceId, fromTs, toTs, ))
                        p.daemon = True
                        p.start()

                        
                        # self.query_results(codec_info, requestID, deviceId, fromTs, toTs)
                        # queryTh = threading.Thread(target=self.query_results, args=(requestID, deviceId, fromTs, toTs, ))
                        # queryTh.daemon = True
                        # queryTh.start()
                        # queryTh.join()
                        
                       
    def query_results(self, codec_info, requestID, deviceId, fromTs, toTs):

        decoder = av.CodecContext.create(codec_info.name,'r')
        decoder.width = codec_info.width
        decoder.height = codec_info.height
        decoder.pix_fmt = codec_info.pix_fmt
        decoder.extradata = codec_info.extradata # important for decoding (PPS, SPS)
        decoder.thread_type = 'AUTO'

        # print("Available filters: ", av.filter.filters_available)
        # settings default memory scaling grpah for in memory queue
        graph = av.filter.Graph()
        fchain = [graph.add_buffer(width=codec_info.width, height=codec_info.height, format=codec_info.pix_fmt, name=requestID)]

        fchain.append(graph.add("scale",self.__filter_scale))
        fchain[-2].link_to(fchain[-1])
        
        fchain.append(graph.add('buffersink'))
        fchain[-2].link_to(fchain[-1])

        graph.configure()

        decodedStreamName = RedisInMemoryDecodedImagesPrefix + deviceId + requestID

        iframeStreamName = RedisInMemoryIFrameListPrefix + deviceId
        # this is where we start our query
        queryTs = self.findClosestIFrameTimestamp(iframeStreamName, fromTs)

        print("Starting to decode in-memory GOP: ", deviceId, fromTs, toTs, queryTs)
        streamName = RedisInMemoryQueuePrefix + deviceId
        
        # sanity check for timestampTo
        redis_time = self.__redis_conn.time()
        redis_time = int(redis_time[0] + (redis_time[1] / 1000000)) * 1000
        if toTs > redis_time:
            toTs = redis_time

        firstIFrameFound = False # used when fromTS is before anything in queue at all (so first I-frame picket)
        while True:
            buffer = self.__redis_conn.xread({streamName: queryTs}, count=30)
            if len(buffer) > 0:
                arr = buffer[0]
                inner_buffer = arr[1]
                last = inner_buffer[-1]
                queryTs = last[0] # remember where to query from next

                # check if we've read everything, exit loop
                last = int(queryTs.decode('utf-8').split("-")[0])
                if last >= int(toTs):
                    print("inmemory buffer decoding finished")
                    break

                for compressed in inner_buffer:
                    compressedData = compressed[1]

                    content = {}
                    for key, value in compressedData.items():
                        content[key.decode("utf-8")] = value

                    if content["is_keyframe"].decode('utf-8') == "0" and firstIFrameFound is False:
                        print("First I-Frame found")
                        firstIFrameFound = True
                    
                    if not firstIFrameFound:
                        continue

                    vf = video_streaming_pb2.VideoFrame()
                    vf.ParseFromString(content["data"])

                    frame_buf = io.BytesIO(vf.data)
                    size = frame_buf.getbuffer().nbytes
                    packet = av.Packet(size)
                    frame_buf.readinto(packet)
                    # packet.pts = vf.pts
                    # packet.dts = vf.dts

                    frames = decoder.decode(packet) or () # should be only 1 frame per packet (for video)
                    if len(frames) <= 0:
                        continue

                    self.addToRedisDecodedImage(graph, decodedStreamName, frames, packet)
        # signal finish (None video frame)
        self.addToRedisDecodedImage(graph, decodedStreamName, None, None)


           
    def findClosestIFrameTimestamp(self, streamName, fromTs):
        '''
        Finds the closest timestamp at exact or before the fromTimestamp in a small queue of iframes
        '''
        searchTs = fromTs

        min = sys.maxsize

        all_i_frames = self.__redis_conn.xread({streamName:0}) # read all in queue
        if len(all_i_frames) > 0:
            all = all_i_frames[0]
            if len(all) > 1:
                iframe_timestamps = all[1]
                for (i, iframe_ts) in enumerate(iframe_timestamps):
                    its = str(iframe_ts[0], 'utf-8')
                    ts = int(its.split("-")[0])

                    if i == 0:
                        searchTs = its
                        continue
                    
                    if ts >= int(fromTs): # stop search (we want only I-frame before fromTs)
                        break

                    # we're always looking for an iframe before fromTs
                    min_abs_candidate = abs(int(fromTs) - ts)
                    if min_abs_candidate < min:
                        searchTs = its
                        min = min_abs_candidate

        # (- 1 ms since xread is exclusive)
        splitted = searchTs.split("-")
        ts = splitted[0]
        tsPart = splitted[1]
        print("found key frame: ", ts, tsPart)
        return str(int(ts)-1) + "-" + tsPart
        
    def addToRedisDecodedImage(self, graph, streamName, frames, packet):
        if frames is None: # signal finish of in-memory buffer read
            vf = video_streaming_pb2.VideoFrame()
            vfData = vf.SerializeToString()
            self.pushDecodedToRedis(streamName, vfData)
            return

        # push decoded frames to redis to be read by server and served back through GRPC
        for frame in frames:
            graph.push(frame)
            
            keepPulling = True
            while keepPulling:
                try:
                    frame = graph.pull()
                    img = frame.to_ndarray(format='bgr24')
                    shape = img.shape

                    img_bytes = np.ndarray.tobytes(img)
                    
                    timestamp = int(time.time() * 1000)
                    if packet.pts is not None and packet.time_base is not None:
                        timestamp = int(packet.pts * float(packet.time_base))

                    vf = video_streaming_pb2.VideoFrame()
                    vf.data = img_bytes
                    vf.width = frame.width
                    vf.height = frame.height
                    vf.timestamp = timestamp
                    vf.frame_type = frame.pict_type.name
                    if packet.pts:
                        vf.pts = packet.pts
                    if packet.dts:
                        vf.dts = packet.dts
                    if packet.time_base is not None:
                        vf.time_base = float(packet.time_base)
                    vf.is_keyframe = packet.is_keyframe
                    vf.is_corrupt = packet.is_corrupt

                    for (i,dim) in enumerate(shape):
                        newDim = video_streaming_pb2.ShapeProto.Dim()
                        newDim.size = dim
                        newDim.name = str(i)
                        vf.shape.dim.append(newDim)

                    vfData = vf.SerializeToString()
                    self.pushDecodedToRedis(streamName, vfData)
                except Exception as e:
                    keepPulling = False

    def pushDecodedToRedis(self, streamName, vfData):
         # in case reading is slow, then this waits until some memory is freed
        # this is due to raw images being stored in memory (e.g. 800x600 RGB would be 4.3MB approx per image)
        started_check = int(time.time() * 1000)
        while True:
            # safety - if reading takes really long (more than 10 seconds, then exit this immidately)
            current_check = int(time.time() * 1000)
            if current_check - started_check > (1000 * 10):
                break

            cnt = self.__redis_conn.xlen(streamName)
            if cnt >= 10:
                time.sleep(0.1)
            else:
                break

        self.__redis_conn.xadd(streamName, {'data': vfData}, maxlen=10)