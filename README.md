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
    * [Install](#install)

## Prerequisites

- [Docker](https://docs.docker.com/engine/install/)
- [Docker Compose](https://docs.docker.com/compose/install/)

Pull `rtsp_to_rtmp` docker image from dockerhub to your local computer:
```bash
docker pull chryscloud/chrysedgeproxy:latest
```

### Enable docker TCP socket connection (Linux, Ubuntu 18.04 LTS)

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

### Install

Build video-edge-ai-proxy:

```bash
docker-compose build
```

Run:
```bash
docker-compose -d up
```

Open browser and visit `http://localhost`

Insert an RTSP camera
TBD: image


# TODO

- [ ] Finish documentation
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