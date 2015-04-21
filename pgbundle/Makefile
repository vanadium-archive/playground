PATH := $(PATH):$(V23_ROOT)/third_party/cout/node/bin
SHELL := /bin/bash -euo pipefail

node_modules: package.json
	@npm prune
	@npm install
	@touch $@

.PHONY: clean
clean:
	@$(RM) -rf node_modules
	@$(RM) -rf npm-debug.log

.PHONY: lint
lint:
	@jshint .
