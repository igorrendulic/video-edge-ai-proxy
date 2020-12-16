#!/bin/sh

# The model here is YOLOv3 model trained by the official
# authors of the model using the DarkNet Framework
# and is made available from their website
# http://pjreddie.com/yolo/

echo 'Getting the YOLOv3 model'
echo 'Starting Download...'
wget -O yolov3-coco/yolov3.weights --no-check-certificate https://pjreddie.com/media/files/yolov3.weights
echo 'Download completed successfully'