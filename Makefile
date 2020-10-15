.PHONY: install server client examples

all: server client examples

server:
	@echo "--> Generating go files"
	protoc -I proto/ --go_out=plugins=grpc:server/proto/ proto/video_streaming.proto
	@echo ""

client:
	@echo "--> Generating Python client files"
	python3 -m grpc_tools.protoc -I proto/ --python_out=python/proto --grpc_python_out=python/proto proto/video_streaming.proto
	@echo ""

examples:
	@echo "--> Generating Python Proto example files"
	python3 -m grpc_tools.protoc -I proto/ --python_out=examples/ --grpc_python_out=examples/ proto/video_streaming.proto
	@echo ""

install: 
	@echo "--> Installing Go and Python grpcio tools"
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	pip3 install -U grpcio grpcio-tools
	@echo ""
