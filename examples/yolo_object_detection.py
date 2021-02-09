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

import numpy as np
import argparse
import imutils
import video_streaming_pb2_grpc, video_streaming_pb2
import time
import cv2 as cv
import os
import subprocess
import grpc

# default values

confidence = 0.5
threshold = 0.3


# os.environ['DISPLAY'] = ":0"

def gen_image_request(device_name, keyframe_only):
    """ Create an object to request a video frame """

    req = video_streaming_pb2.VideoFrameRequest()
    req.device_id = device_name
    req.key_frame_only = keyframe_only
    return req

def draw_labels_and_boxes(img, boxes, confidences, classids, idxs, colors, labels):
    # If there are any detections
    if len(idxs) > 0:
        for i in idxs.flatten():
            # Get the bounding box coordinates
            x, y = boxes[i][0], boxes[i][1]
            w, h = boxes[i][2], boxes[i][3]
            
            # Get the unique color for this class
            color = [int(c) for c in colors[classids[i]]]

            # Draw the bounding box rectangle and label on the image
            cv.rectangle(img, (x, y), (x+w, y+h), color, 2)
            text = "{}: {:4f}".format(labels[classids[i]], confidences[i])
            cv.putText(img, text, (x, y-5), cv.FONT_HERSHEY_SIMPLEX, 0.5, color, 2)

    return img

def generate_boxes_confidences_classids(outs, height, width, tconf):
    boxes = []
    confidences = []
    classids = []

    for out in outs:
        for detection in out:
            #print (detection)
            #a = input('GO!')

            # Get the scores, classid, and the confidence of the prediction
            scores = detection[5:]
            classid = np.argmax(scores)
            confidence = scores[classid]

            # Consider only the predictions that are above a certain confidence level
            if confidence > tconf:
                # TODO Check detection
                box = detection[0:4] * np.array([width, height, width, height])
                centerX, centerY, bwidth, bheight = box.astype('int')

                # Using the center x, y coordinates to derive the top
                # and the left corner of the bounding box
                x = int(centerX - (bwidth / 2))
                y = int(centerY - (bheight / 2))

                # Append to list
                boxes.append([x, y, int(bwidth), int(bheight)])
                confidences.append(float(confidence))
                classids.append(classid)

    return boxes, confidences, classids

def infer_image(net, layer_names, height, width, img, colors, labels, boxes=None, confidences=None, classids=None, idxs=None, infer=True):
    
    if infer:
        # Contructing a blob from the input image
        blob = cv.dnn.blobFromImage(img, 1 / 255.0, (416, 416), swapRB=True, crop=False)

        # Perform a forward pass of the YOLO object detector
        net.setInput(blob)

        # Getting the outputs from the output layers
        start = time.time()
        outs = net.forward(layer_names)
        end = time.time()

        print ("[INFO] YOLOv3 took {:6f} seconds".format(end - start))
        
        # Generate the boxes, confidences, and classIDs
        boxes, confidences, classids = generate_boxes_confidences_classids(outs, height, width, confidence)
        
        # Apply Non-Maxima Suppression to suppress overlapping bounding boxes
        idxs = cv.dnn.NMSBoxes(boxes, confidences, confidence, threshold)

    if boxes is None or confidences is None or idxs is None or classids is None:
        raise '[ERROR] Required variables are set to None before drawing boxes on images.'
        
    # Draw labels and boxes on the image
    img = draw_labels_and_boxes(img, boxes, confidences, classids, idxs, colors, labels)

    return img, boxes, confidences, classids, idxs


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Chrysalis Edge Proxy Basic Example')
    parser.add_argument("--device", type=str, default=None, required=True)
    parser.add_argument("--keyframe", action='store_true')
    args = parser.parse_args()

    # grpc connection to video-edge-ai-proxy
    options = [('grpc.max_receive_message_length', 50 * 1024 * 1024)]
    channel = grpc.insecure_channel('127.0.0.1:50001', options=options)
    stub = video_streaming_pb2_grpc.ImageStub(channel)

    print("starting YoloV3 COCO object detection")
    
    # Download the YOLOv3 models if needed
    if not os.path.exists("./yolov3-coco/yolov3.weights"):
        subprocess.call(['./yolov3-coco/get-model.sh'])

    # Get the labels
    labels = open("yolov3-coco/coco-labels.txt").read().strip().split('\n')

    # Intializing colors to represent each label uniquely
    colors = np.random.randint(0, 255, size=(len(labels), 3), dtype='uint8')

    # Load the weights and configutation to form the pretrained YOLOv3 model
    net = cv.dnn.readNetFromDarknet("yolov3-coco/yolov3.cfg", "yolov3-coco/yolov3.weights")
    
    # Get the output layer names of the model
    layer_names = net.getLayerNames()
    layer_names = [layer_names[i[0] - 1] for i in net.getUnconnectedOutLayers()]

    count = 0

    boxes = None
    confidences = None
    classids = None
    idxs = None
    
    while True:
        prev = int(time.time() * 1000)
        frame = stub.VideoLatestImage(gen_image_request(device_name=args.device,keyframe_only=args.keyframe))
        # read raw frame data and convert to numpy array
        img_bytes = frame.data 
        re_img = np.frombuffer(img_bytes, dtype=np.uint8)

        # reshape image back into original dimensions
        if len(frame.shape.dim) > 0:
            reshape = tuple([int(dim.size) for dim in frame.shape.dim])
            re_img = np.reshape(re_img, reshape)

            height, width = re_img.shape[:2]

            re_img, boxes, confidences, classids, idxs = infer_image(net, layer_names, \
                                height, width, re_img, colors, labels, boxes, confidences, classids, idxs, infer=True)

            # add camera name
            cv.namedWindow('box', cv.WINDOW_NORMAL)
            cv.resizeWindow('box', 1024,768)
            cv.setWindowTitle('box', args.device) 
            cv.imshow('box', re_img)
            
            if cv.waitKey(1) & 0xFF == ord('q'):
                break

cv2.destroyWindow('box')