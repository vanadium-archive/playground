PATH := node_modules/.bin:$(PATH)
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

.PHONY: dependency-check
dependency-check: package.json node_modules
	dependency-check $<

.PHONY: lint
lint: dependency-check
	jshint .
