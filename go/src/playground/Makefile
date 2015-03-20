PATH := bin:node_modules/.bin:$(PATH)
PATH := $(PATH):$(VANADIUM_ROOT)/environment/cout/node/bin
PATH := $(PATH):$(VANADIUM_ROOT)/release/projects/playground/go/bin
SHELL := /bin/bash -euo pipefail
export GOPATH := $(VANADIUM_ROOT)/release/projects/playground/go:$(GOPATH)
export VDLPATH := $(GOPATH)

host ?= 127.0.0.1
port ?= 8181

.DELETE_ON_ERROR:

.PHONY: all
all:
	v23 go install playground/...

.PHONY: all
start: config/db.json
	compilerd \
		--sqlconf=$< \
		--setupdb=true \
		--listenTimeout=0 \
		--address=$(host):$(port) \
		--use-docker=false

config/db.json:
	@echo "You are missing config/db.json, create this file based"
	@echo "on the appropriate examples in the config directory."
	@exit 1;

.PHONY: test
test:
	v23 run ./test.sh
