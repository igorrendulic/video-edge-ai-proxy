#!/bin/bash

# stop bash script on error
# set -o errexit
set -o noclobber # enable >|
set -e

rtsp_endpoint=${rtsp_endpoint}
device_id=${device_id}
rtmp_endpoint=${rtmp_endpoint}
in_memory_buffer=${in_memory_buffer}
in_memory_scale=${memory_scale}
disk_buffer_path=${disk_buffer_path}
disk_cleanup_rate=${disk_cleanup_rate}
redis_host=${redis_host}
redis_port=${redis_port}

 if [ -z "$rtsp_endpoint" ]; then 
    echo "rtsp endpoint must be defined in environment variables"
    exit 1 
fi
if [ -z "$device_id" ]; then 
    echo "device_id endpoint must be defined in environment variables"
    exit 1
fi


echo "Connection to rtsp camera"
source activate chrysedgeai

cmd=" -u rtsp_to_rtmp.py --rtsp $rtsp_endpoint --device_id $device_id"
if [ ! -z "$rtmp_endpoint" ]; then 
    cmd="$cmd --rtmp $rtmp_endpoint"
fi
if [ ! -z "$in_memory_buffer" ]; then
    cmd="$cmd --memory_buffer $in_memory_buffer"
fi
if [ ! -z "$in_memory_scale" ]; then
    cmd="$cmd --memory_scale $in_memory_scale"
fi
if [ ! -z "$disk_buffer_path" ]; then
    cmd="$cmd --disk_path $disk_buffer_path"
fi
if [ ! -z "$disk_cleanup_rate" ]; then
    cmd="$cmd --disk_cleanup_rate $disk_cleanup_rate"
fi
if [ ! -z "$redis_host" ]; then
    cmd="$cmd --redis_host $redis_host"
fi
if [ ! -z "$redis_port" ]; then
    cmd="$cmd --redis_port $redis_port"
fi

echo "running: $cmd"

python $cmd

echo $?
echo "Cant start rtsp_to_rtmp.py. Exiting..."