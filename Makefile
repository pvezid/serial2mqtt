TOP := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

build:
	@(cd $(TOP)/src; go build -o $(TOP)/bin/ cmd/messager.go)
	@(cd $(TOP)/src; go build -o $(TOP)/bin/ cmd/serial2mqtt.go)

install: build
	sudo install -s $(TOP)/bin/messager /usr/local/bin

mrproper:
	@(rm -rf $(TOP)/bin)
