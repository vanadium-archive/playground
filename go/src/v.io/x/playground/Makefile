PATH := $(JIRI_ROOT)/release/projects/playground/go/bin:$(PATH)
SHELL := /bin/bash -euo pipefail
export GOPATH := $(JIRI_ROOT)/release/projects/playground/go:$(GOPATH)
export VDLPATH := $(JIRI_ROOT)/release/projects/playground/go/src:$(VDLPATH)

host ?= 127.0.0.1
port ?= 8181

.DELETE_ON_ERROR:

.PHONY: all
all: builder compilerd

# builder-deps target builds the binaries that the builder calls.
.PHONY: builder-deps
builder-deps:
	jiri go install v.io/x/ref/cmd/principal
	jiri go install v.io/x/ref/services/identity/identityd
	jiri go install v.io/x/ref/services/mounttable/mounttabled
	jiri go install v.io/x/ref/services/xproxy/xproxyd

.PHONY: builder
builder: builder-deps
	jiri go install v.io/x/playground/builder

.PHONY: compilerd
compilerd:
	jiri go install v.io/x/playground/compilerd

.PHONY: start
start: config/db.json all
	compilerd \
		--sqlconf=$< \
		--listen-timeout=0 \
		--address=$(host):$(port) \
		--origin='*' \
		--use-docker=false

.PHONY: pgadmin
pgadmin:
	jiri go install v.io/x/playground/pgadmin

.PHONY: updatedb
updatedb: config/db.json pgadmin
	pgadmin \
		--sqlconf=$< \
		migrate up

.PHONY: bootstrap
bootstrap: config/db.json pgadmin
	pgadmin \
		--sqlconf=$< \
		bundle bootstrap

config/db.json:
	@echo "You are missing config/db.json, create this file based"
	@echo "on the appropriate examples in the config directory."
	@exit 1;

# Temporary workaround for linking Go tests into Jenkins.
.PHONY: test
test:
	jiri go test v.io/x/playground/builder -v23.tests
