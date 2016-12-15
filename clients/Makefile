# makefile for solidity clients

default: all

all: clients

clients: build_bootstrapv2 build_sync build_dir_test build_agp

build_bootstrapv2: bootstrapv2/bootstrapv2.go
	cd bootstrapv2; make

build_sync: sync/wait_sync.go
	cd sync; make

build_dir_test: directory_test/test_directory.go
	cd directory_test; make

build_agp: agreement_protocol/agreement_protocol.go
	cd agreement_protocol; make
