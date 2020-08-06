# video-edge-ai-proxy display example

Displays Video Stream from video-edge-ai-proxy with OpenCV

## Prerequsities

[Install Conda](https://docs.conda.io/projects/conda/en/latest/user-guide/install/)

Run `docker-compose -d up` in the main folder of the project

## Install

Create conda environment with OpenCV:
```
conda env create -f environment.yml
```

Activate environment:
```
conda activate chrysedgedisplay
```

## Usage

Example script uses a `grpc` interface to communicate with the server. 

List if any camera connected:
```
python client_example.py --list true
```

Expected output:
```
--------------------DEVICES---------------------
name: "test"
status: "running"
pid: 28455
running: true
```

Display video in OpenCV:
```
python client_example --device test
```



