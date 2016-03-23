NODE_BIN := $(shell jiri profile env --profiles=v23:base,v23:nodejs NODE_BIN=)
PATH := bin:node_modules/.bin:$(PATH):$(NODE_BIN)
SHELL := /bin/bash -euo pipefail

# NOTE: we run npm using 'node npm' to avoid relying on the shebang line in the
# npm script, which can exceed the Linux shebang length limit on Jenkins.
NPM := $(NODE_BIN)/npm

js_files := $(shell find browser -name "*.js")
css_files := $(shell find stylesheets -name "*.css")
host ?= 127.0.0.1
port ?= 8088

.DEFAULT_GOAL := all

.DELETE_ON_ERROR:

.PHONY: all
all: public/bundle.js public/bundle.css
	@true  # silences `watch make`

.PHONY: deploy-production
deploy-production: all
	make -C $(JIRI_ROOT)/infrastructure/deploy playground-production

.PHONY: deploy-staging
deploy-staging: all
	make -C $(JIRI_ROOT)/infrastructure/deploy playground-staging

public/bundle.js: browser/index.js $(js_files) node_modules
	browserify --debug $< 1> $@

public/bundle.css: stylesheets/index.css $(css_files) node_modules
	bin/compile-css $< 1> $@

node_modules: package.json
	@node $(NPM) prune
	@node $(NPM) install
	@touch $@

.PHONY: start
start: all
	http-server public -p $(port) -a $(host) -d --cors

.PHONY: clean
clean:
	@$(RM) -rf public/bundle.*
	@$(RM) -rf node_modules
	@$(RM) -rf npm-debug.log

.PHONY: distclean
distclean: clean
	@node $(NPM) cache clean

.PHONY: dependency-check
dependency-check: package.json node_modules
	dependency-check $<

.PHONY: lint
lint: dependency-check
	jshint .

.PHONY: test
test: lint test-client test-browser
	@true  # silences `watch make`

.PHONY: test-client
test-client: node_modules
	tape test/index.js

.PHONY: test-browser
test-browser: node_modules
	run-browser test/index.js --phantom
