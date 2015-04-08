PATH := bin:node_modules/.bin:$(PATH)
PATH := $(PATH):$(V23_ROOT)/environment/cout/node/bin
PATH := $(V23_ROOT)/release/projects/playground/go/bin:$(PATH)
SHELL := /bin/bash -euo pipefail
export GOPATH := $(V23_ROOT)/release/projects/playground/go:$(GOPATH)
export VDLPATH := $(GOPATH)

host ?= 127.0.0.1
port ?= 8181

.DELETE_ON_ERROR:

.PHONY: all
all: builder compilerd

# builder-deps target builds the binaries that the builder calls.
.PHONY: builder-deps
builder-deps:
	v23 go install v.io/x/ref/cmd/principal
	v23 go install v.io/x/ref/services/identity/identityd
	v23 go install v.io/x/ref/services/mounttable/mounttabled
	v23 go install v.io/x/ref/services/proxy/proxyd
	v23 go install v.io/x/ref/services/wspr/wsprd

.PHONY: builder
builder: builder-deps
	v23 go install playground/builder

.PHONY: compilerd
compilerd:
	v23 go install playground/compilerd

.PHONY: start
start: config/db.json all
	compilerd \
		--sqlconf=$< \
		--setupdb=true \
		--listen-timeout=0 \
		--address=$(host):$(port) \
		--use-docker=false

config/db.json:
	@echo "You are missing config/db.json, create this file based"
	@echo "on the appropriate examples in the config directory."
	@exit 1;

.PHONY: test
test:
	v23 run ./test.sh
