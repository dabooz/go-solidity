# makefile for solidity clients

default: all

all: clients

clients: build_bootstrap build_device build_cp build_sync build_rest_cp build_dir_test build_deregister

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

build_dir_test: directory_test/test_directory.go
	cd directory_test; make

build_deregister: deregister/deregister.go
	cd deregister; make