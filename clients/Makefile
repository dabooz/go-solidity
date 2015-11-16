# makefile for solidity clients

default: all

all: clients

clients: build_bootstrap build_device build_cp build_sync build_rest_cp

build_bootstrap: bootstrap/bootstrap.go
	cd bootstrap; make

build_device: device/device_owner.go
	cd device; make

build_cp: container_provider/container_provider.go
	cd container_provider; make

build_sync: sync/wait_sync.go
	cd sync; make

build_rest_cp: rest_container_provider/rest_container_provider.go
	cd rest_container_provider; make