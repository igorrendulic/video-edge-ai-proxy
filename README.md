# video-edge-ai-proxy

**Currently this repository is under active development!**

The ultimate video pipeline for Computer Vision.

Video Edge-AI Proxy ingests multiple RTSP camera streams and provides a common interface for conducting AI operations on or near the Edge.

## Why use Video Edge-AI Proxy?

video-edge-ai-proxy is an easy to use collection mechanism from multiple cameras onto a single more powerful computer. For example, a network of CCTV RTSP enabled cameras can be accessed through a simple GRPC interface, where Machine Learning algorithms can do various Computer Vision tasks. Furthermore, interesting footage can be annotated, selectively streamed and stored through a simple API for later analysis, computer vision tasks in the cloud or enriching the Machine Learning training samples.

<p align="center">
    <img src="https://storage.googleapis.com/chrysaliswebassets/chrysalis-video-edge-ai-proxy.png" title="Chrysalis Cloud" />
<p align="center">

## Features

- **RTSP camera hub** - User interface and RESTful API for setting up multiple RTSP cameras
- **Connection management** - handles cases of internet outages or camera streaming problems
- **Stream management** - deals with the complexities of stream management
- **Video/Image Hub** - processing of images from multiple camera sources simultaneously
- **Optimized** - optimized for processing multiple camera streams in parallel
- **Selective Frames** - can read I-Frames (Keyframes) or Frames within any time interval (skipping decoding of packets when possible)
- **Selective Pass-Through** - selective streamin (start/stop) for preserving bandwidth
- **Selective Pass-Through-Storage** - on and off switch for storing a portion of a stream forwarded to Chrysalis Cloud
- **Machine Learning Annotation** - asynchronous annotations for live video streams

## Contents

* [Prerequisites](#prerequisites)
* [Quick Start](#quick-start)
* [Portal usage](#portal-usage)
* [Client usage](#client-usage)
* [Examples](#examples)
  * [Prerequisites](#example-prerequisites)
  * [Running basic_usage.py](#example-prerequisites)
  * [Running opencv_display.py](#example-prerequisites)
  * [Running annotation.py](#example-prerequisites)
  * [Running storage_onoff.py](#example-prerequisites)
* [Custom configuration](#custom-configuration)
  * [Custom Redis Configuration](#custom-redis-configuration)
  * [Custom Chrysalis Configuration](#custom-chrysalis-configuration)
* [Building from source](#building-from-source)

## Prerequisites

- [Docker](https://docs.docker.com/engine/install/)
- [Docker Compose](https://docs.docker.com/compose/install/)

Pull `rtsp_to_rtmp` docker image from dockerhub to your local computer:
```bash
docker pull chryscloud/chrysedgeproxy:0.0.2
```

#### Enable docker TCP socket connection

***Linux based systems only***

`This settings are not required if you running on Mac OS X and Windows. Only make sure that docker-compose and docker are updated to the latest versions`.

Create `daemon.json` file in `/etc/docker` folder with JSON contents:
```json
{
  "hosts": [
    "fd://",
    "unix:///var/run/docker.sock"
  ]
}
```

Create a new file `/etc/systemd/system/docker.service.d/docker.conf` with the following contents:
```
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd
```

Reload daemon:
```
sudo systemctl daemon-reload
```

Restart docker
```
sudo service docker restart
```

You can test out if the configuration is correct by issuing curl request docker socket:
```
curl -s --unix-socket /var/run/docker.sock http://dummy/images/json | jq '.'
```

## Quick Start

By default video-edge-ai-proxy requires these ports:
- *80* for web portal
- *8080* for RESTful API (portal API)
- *50001* for client grpc connection
- *6379* for redis

Make sure before your run it that these ports are available.

For **Mac OS X and Windows 10** update your Docker desktop to latest version.

If running on **Mac OS X** make sure to modify `/data/chrysalis` to a more Mac OS friendly folder e.g. `/Users/usename/data` under `chrysedgeserver` -> `volumes`. 

If running on **Windows 10** make sure to modify `/data/chrysalis` to your custom windows folder to e.g. `c:/Users/user/chrys-video-egde-proxy/data` under `chrysedgeserver` -> `volumes`. Create the folder first. On Windows latest Docker Desktop version must have  `WSL integration` enabled in settings. 

Create a directory: `/data/chrysalis` or for Mac OS X: `/Users/usename/data`

Copy and paste `docker-compose.yml` to folder of your choice (recommended to be different than /data/chrysalis):

```yml
version: '3.8'
services:
  chrysedgeportal:
    image: chryscloud/chrysedgeportal:0.0.5
    depends_on:
      - chrysedgeserver
      - redis
    ports:
      - "80:80"
    networks:
      - chrysnet
  chrysedgeserver:
    image: chryscloud/chrysedgeserver:0.0.5
    restart: always
    depends_on:
      - redis
    entrypoint: /app/main
    ports:
      - "8080:8080"
      - "50001:50001"
    volumes:
      - /data/chrysalis:/data/chrysalis
      - /var/run/docker.sock:/var/run/docker.sock
    networks: 
      - chrysnet
  redis:
    image: "redis:alpine"
    ports:
      - "6379:6379"
    # volumes:
    #   - /data/chrysalis/redis:/data
    #   - ./redis.conf:/usr/local/etc/redis/redis.conf
    # command:
    #   - redis-server
    #   - /usr/local/etc/redis/redis.conf
  
    networks: 
      - chrysnet

networks:
  chrysnet:
    name: chrysnet
```

Start video-edge-ai-proxy:
```bash
docker-compose pull
docker-compose up -d --no-build
```

(Currently H.264 support only)

Open browser and visit `chrysalisportal` at address: `http://localhost`

## Portal usage

Open your browser and go to: `http://localhost`.

On the first visit Edge Proxy will display a RTSP docker container icon. Click on it. This will initiate the pull for the latest version of the docker container pre-compiled to be used with RTSP enabled cameras. 

Connecting RTSP camera

1. Click: `Connect RTSP Camera` in the `chrysalisportal` and name the camera (e.g. `test`)
2. Insert full RTSP link (if credentials are required then add them to the link)

Example RTSP url: `rtsp://admin:12345@192.168.1.21/Streaming/Channels/101` where admin is username and 12345 is the password.

Example RTSP url: `rtsp://192.168.1.21:8554/unicast` when no credentials required and non-default port.

Click on the newly created connection and check the output and error log. Expected state is `running` and output `Started python rtsp process...`

<p align="center">
    <img src="https://storage.googleapis.com/chrysaliswebassets/chrys_edge_proxy_test_cam.png" title="Chrysalis Edge Proxy test cam" />
<p align="center">

We're ready to consume frames from RTSP camera. Check the `/examples` folder.

## Client usage

At this point you should have the video-edge-ai-proxy up and running and your first connection to RTSP camera made.

## Examples

### Example Prerequisites

Create conda environment:
```
conda env create -f examples/environment.yml
```

Activate environment:
```
conda activate chrysedgeexamples
cd examples
```

Generate python grpc stubs:
```
make examples
```

### Running `basic_usage.py`

List all stream processes:
```
python basic_usage.py --list
```

Successful output example:
```
name: "test"
status: "running"
pid: 18109
running: true
```

Output single streaming frame information from `test` camera:
```
python basic_usage.py --device test
```

Successful output example:
```
is keyframe:  False
frame type:  P
frame shape:  dim {
  size: 480
  name: "0"
}
dim {
  size: 640
  name: "1"
}
dim {
  size: 3
  name: "2"
}
```

- is_keyframe (True/False)
- frame type: (I,P,B)
- frame shape: image dimensions (always in BGR24 format). In this example: `480x640x3 bgr24`


### Running `opencv_display.py`

Display video at original frame rate for `test` camera:
```
python opencv_display.py --device test
```

Display only Keyframes for `test` camera:
```
python opencv_display.py --device test --keyframe
```


### Running `annotation.py`

Asynchronous annotation from the edge.

```
python annotation.py --device test --type thisistest
```


### Running `storage_onoff.py`

Storage example turn Chrysalis Cloud storage on or off for the current live stream from the cameras. 

Run example to turn storage on for camera `test`:
```
python storage_onoff.py --device test --on true
```

Run example to turn storage off for camera `test`:
```
python storage_onoff.py --device test --on false
```

# Custom configuration

## Custom redis configuration

Default configuration is in the root folder of this project: `./redis.conf`

1. Update default `redis.conf` in the root directory of this project
2. Uncomment volumes section in redis config
```yaml   
    # volumes:
    #   - /data/chrysalis/redis:/data
    #   - ./redis.conf:/usr/local/etc/redis/redis.conf
    # command:
    #   - redis-server
    #   - /usr/local/etc/redis/redis.conf
```

Modify folders accordingly for **Mac OS X and Windows**

## Custom Chrysalis configuration

Create `conf.yaml` file in `/data/chrysalis` folder. The configuration file is automatically picked up if it exists otherwise system fallbacks to it's default configuration.

```yaml
version: 0.0.3
title: Chrysalis Video Edge Proxy
description: Chrysalis Video Edge Proxy Service for Computer Vision
mode: release # "debug": or "release"

redis:
  connection: "redis:6379"
  database: 0
  password: ""

api:
  endpoint: https://api.chryscloud.com

annotation:
  endpoint: "https://event.chryscloud.com/api/v1/annotate"
  unacked_limit: 1000
  poll_duration_ms: 300
  max_batch_size: 299

buffer:
  in_memory: 1 # number of images to store in memory buffer (1 = default)
  on_disk: false # store key-frame separated mp4 file segments to disk
  on_disk_folder: /data/chrysalis/archive # can be any custom folder you'd like to store video segments to
  on_disk_clean_older_than: "5m" # remove older mp4 segments than 5m
```

- `mode: release`: disables debug mode for http server (default: release)
- `redis -> connection`: redis host with port (default: "redis:6379")
- `redis -> database` : 0 - 15. 0 is redis default database. (default: 0)
- `redis -> password`: optional redis password (default: "")
- `api -> endpoint`: chrysalis API location for remote signaling such as enable/disable storage (default: https://api.chryscloud.com)
- `annotation -> endpoint`: Crysalis Cloud annotation endpoint (default: https://event.chryscloud.com/api/v1/annotate)
- `annotation -> unacked limit`: maximum number of unacknowledged annotatoons (default: 299)
- `annotation -> poll_duration_ms`: poll every x miliseconds for batching purposes (default: 300ms)
- `annotation -> max_match_size`: maximum number of annotation per batch size (default: 299)
- `buffer -> in_memory`: number of decoded frames to store in memory per camera (default: 1)
- `on_disk`: true/false, store key-frame chunked mp4 files to disk (default: false)
- `on_disk_folder`: path to the folder where segments will be stored
- `on_disk_clean_older_than`: remove mp4 segments older than (default: 5m)
- `on_disk_schedule`: run disk cleanup scheduler cron job [#https://en.wikipedia.org/wiki/Cron](https://en.wikipedia.org/wiki/Cron)

`on_disk` creates mp4 segments in format: `"current_timestamp in ms"_"duration_in_ms".mp4`. For example: `1600685088000_2000.mp4`

If running on **Mac OS X** modify `on_disk_folder` to your custom one. 

If running on **Windows 10** modify `on_disk_folder` by prefixing `/C/`. Example:
```
on_disk_folder:  /C/Users/user/chrys-video-egde-proxy/videos
```

### Building from source

```
git clone https://github.com/chryscloud/video-edge-ai-proxy.git
```

video-edge-ai-proxy stores running processes (1 for each connected camera) into a local datastore hosted on your file system. By default the folder path used is:
- */data/chrysalis*

Create the folder if it doesn't exist and make sure it's writtable by docker process.

In case you cloned this repository you can run docker-compose with build command. 
`Start video-edge-ai-proxy` with local build:
```bash
docker-compose up -d
```

or 
```
docker-compose build
```



# RoadMap

- [X] Finish documentation
- [X] Configuration (custom configuration)
- [X] Set enable/disabled flag for storage
- [X] Add API key to Chrysalis Cloud for enable/disable storage
- [X] Add configuration for in memory buffer pool of decoded image so they can be queried in the past
- [X] Configuration and a cron job to store mp4 segments (1 per key-frame) from cameras and a cron job to clean old mp4 segments (rotating file buffer)
- [ ] Add gRPC API to query in-memory buffer of images
- [ ] Remote access Security (grpc TLS Client Authentication)
- [ ] Remote access Security (TLS Client Authentication for web interface)
- [ ] add RTMP container support (mutliple streams, same treatment as RTSP cams)
- [ ] add v4l2 container support (e.g. Jetson Nano, Raspberry Pi?)
- [ ] Initial web screen to pull images (RTSP, RTMP, V4l2)
- [ ] Benchmark NVDEC,NVENC, VAAPI hardware decoders

# Contributing

Please read `CONTRIBUTING.md` for details on our code of conduct, and the process of submitting pull requests to us. 

# Versioning

Current version is initial release - v0.0.1 prerelease

# Authors

- **Igor Rendulic** - Initial work - [Chrysalis Cloud](https://chryscloud.com)

# License

This project is licensed under Apache 2.0 License - see the `LICENSE` for details.


# Acknowledgments

<p align="left">
    <img src="https://storage.googleapis.com/chrysaliswebassets/logo_small.png" style="width: 300px;" title="Chrysalis Cloud" />



