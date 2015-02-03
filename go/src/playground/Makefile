PATH := bin:node_modules/.bin:$(PATH)
PATH := $(PATH):$(VANADIUM_ROOT)/environment/cout/node/bin
SHELL := /bin/bash -euo pipefail
export GOPATH := $(realpath $(shell pwd)/../..):$(GOPATH)
export VDLPATH := $(GOPATH)

.DELETE_ON_ERROR:

.PHONY: all
all:
	v23 go install playground/...

.PHONY: test
test:
	v23 run ./test.sh
