PATH := $(PATH):$(VANADIUM_ROOT)/environment/cout/node/bin
SHELL := /bin/bash -euo pipefail

node_modules: package.json
	@npm prune
	npm install
	@touch node_modules

.PHONY: clean
clean:
	@$(RM) -rf node_modules

.PHONY: lint
lint:
	@jshint .
