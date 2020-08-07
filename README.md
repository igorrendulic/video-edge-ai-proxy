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
- **Video/Image Hub** - processing of images from multiple camera sources simulaneously
- **Optimized** - optimized for processing multiple camera streams in parallel
- **Selective Frames** - can read I-Frames (Keyframes) or Frames within any time interval (optimized stream decoding)
- **Selective Pass-Through** - selective streamin (start/stop) for preserving bandwidth
- **Selective Pass-Through-Storage** - on and off switch for storing a portion of a stream forwarded to Chrysalis Cloud
- **Machine Learning Annotation** - annotating live video streams

## Contents

* [Prerequisites](#prerequisites)
* [Quick Start](#quick-start)
    * [How to run](#how-to-run)
* [Portal usage](#portal-usage)
* [Client usage](#client-usage)
* [Examples](#examples)
  * [Example Prerequisites](#example-prerequisites)
  * [Running basic_usage.py](#running-basic_usage.py)
  * [Running opencv_display.py](#running-opencv_display.py)
* [Build](#build)

## Prerequisites

- [Docker](https://docs.docker.com/engine/install/)
- [Docker Compose](https://docs.docker.com/compose/install/)

Pull `rtsp_to_rtmp` docker image from dockerhub to your local computer:
```bash
docker pull chryscloud/chrysedgeproxy:0.0.1
```

#### Enable docker TCP socket connection (Linux, Ubuntu 18.04 LTS)

Create `daemon.json` file in `/etc/docker` folder and add in:
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

## Quick Start

By default video-edge-ai-proxy requires these ports:
- *80* for web portal
- *8080* for RESTful API (portal API)
- *50001* for client grpc connection
- *6379* for redis

Make sure before your run it that these ports are available.

### How to run

video-edge-ai-proxy stores running processes (1 for each connected camera) into a local datastore hosted on your file system. By default the folder path used is:
- */data/chrysalis*

Create the folder if it doesn't exist and make sure it's writtable by docker process.


`Start video-edge-ai-proxy`:
```bash
docker-compose -d up
```

Open browser and visit `chrysalisportal` at address: `http://localhost`


## Portal usage

Open your browser and go to: `http://localhost`

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

Generate python grpc stubs:
```
make examples
```

Create conda environment:
```
conda env create -f examples/environment.yml
```

Activate environment:
```
conda activate chrysedgeexamples
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


## Build

Building from source code:

```
docker-compose build
```


# RoadMap

- [X] Finish documentation
- [ ] Set enable/disabled flag for storage
- [ ] Bug(r) - occasionaly few packets for decoding skipped when enabling/disabling rtmp stream (visible only if high FPS on display)
- [ ] Add API key to Chrysalis Cloud for enable/disable storage
- [ ] Security (grpc TLS Client Authentication)
- [ ] Security (TLS Client Authentication for web interface)
- [ ] Security (auto generate redis password)
- [ ] Configuration (extract configuration for custom port assingnment)
- [ ] add RTMP container support (mutliple streams, same treatment as RTSP cams)
- [ ] add v4l2 container support (e.g. Jetson Nano, Raspberry Pi?)
- [ ] Initial web screen to pull images (RTSP, RTMP, V4l2)

# Contributing

Please read `CONTRIBUTING.md` for details on our code of conduct, and the process of submitting pull requests to us. 

# Versioning

Current version is initial release - v1.0.0

# Authors

- **Igor Rendulic** - Initial work - [Chrysalis Cloud](https://chryscloud.com)

# License

This project is licensed under Apache 2.0 License - see the `LICENSE` for details.


# Acknowledgments

<p align="left">
    <img src="https://storage.googleapis.com/chrysaliswebassets/logo_small.png" style="width: 300px;" title="Chrysalis Cloud" />



