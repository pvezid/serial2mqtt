TOP := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

build:
	@(cd $(TOP)/src; go build -o $(TOP)/bin/ cmd/messager.go)
	@(cd $(TOP)/src; go build -o $(TOP)/bin/ cmd/serial2mqtt.go)

mrproper:
	@(rm -rf $(TOP)/bin)
