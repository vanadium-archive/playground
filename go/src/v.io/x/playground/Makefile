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
	v23 go install -a -tags wspr v.io/x/ref/services/wspr/wsprd
	v23 go install v.io/x/ref/cmd/principal
	v23 go install v.io/x/ref/services/identity/identityd
	v23 go install v.io/x/ref/services/mounttable/mounttabled
	v23 go install v.io/x/ref/services/proxy/proxyd

.PHONY: builder
builder: builder-deps
	v23 go install v.io/x/playground/builder

.PHONY: compilerd
compilerd:
	v23 go install v.io/x/playground/compilerd

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
	v23 go install v.io/x/playground/pgadmin

.PHONY: updatedb
updatedb: config/db.json pgadmin
	pgadmin \
		--sqlconf=$< \
		migrate up

.PHONY: bootstrap
bootstrap: config/db.json pgadmin pgbundle
	pgadmin \
		--sqlconf=$< \
		bundle bootstrap

config/db.json:
	@echo "You are missing config/db.json, create this file based"
	@echo "on the appropriate examples in the config directory."
	@exit 1;

# TODO(ivanpi): Remove when pgbundle is ported to Go.
.PHONY: pgbundle
pgbundle:
	PATH="$(PATH):$(V23_ROOT)/third_party/cout/node/bin" \
		npm install --production "$(V23_ROOT)/release/projects/playground/pgbundle"

.PHONY: clean
clean:
	@$(RM) -rf node_modules
	@$(RM) -rf npm-debug.log

# Temporary workaround for linking Go tests into Jenkins.
.PHONY: test
test:
	v23 go test v.io/x/playground/builder -v23.tests
