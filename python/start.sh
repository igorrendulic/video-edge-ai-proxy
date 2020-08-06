#!/bin/bash

# stop bash script on error
# set -o errexit
set -o noclobber # enable >|
set -e

rtsp_endpoint=${rtsp_endpoint}
device_id=${device_id}
rtmp_endpoint=${rtmp_endpoint}

 if [ -z "$rtsp_endpoint" ]; then 
    echo "rtsp endpoint must be defined in environment variables"
    exit 1 
fi
if [ -z "$device_id" ]; then 
    echo "device_id endpoint must be defined in environment variables"
    exit 1
fi


echo "Started python rtsp process..."
source activate chrysedgeai
if [ -z "$rtmp_endpoint" ]; then 
    python rtsp_to_rtmp.py --rtsp "$rtsp_endpoint" --device_id "$device_id"    
else
    python rtsp_to_rtmp.py --rtsp "$rtsp_endpoint" --device_id "$device_id" --rtmp "$rtmp_endpoint"
fi

echo $?
echo "Cant start rtsp_to_rtmp.py. Exiting..."