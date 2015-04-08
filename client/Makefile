PATH := bin:node_modules/.bin:$(PATH)
PATH := $(PATH):$(V23_ROOT)/environment/cout/node/bin
SHELL := /bin/bash -euo pipefail
export GOPATH := $(V23_ROOT)/release/projects/playground/go:$(GOPATH)
export VDLPATH := $(GOPATH)

js_files := $(shell find browser -name "*.js")
css_files := $(shell find stylesheets -name "*.css")
host ?= 127.0.0.1
port ?= 8088

.DEFAULT_GOAL := all

.DELETE_ON_ERROR:

.PHONY: all
all: public/bundles public/bundle.js public/bundle.css
	@true  # silences `watch make`

.PHONY: deploy-staging
deploy-staging: all
	git rev-parse --verify HEAD > public/version
	gcloud config set project vanadium-staging
	gsutil -m rsync -d -r public gs://playground.staging.v.io

public/bundle.js: browser/index.js $(js_files) node_modules
	browserify --debug $< 1> $@

# TODO(jasoncampbell): Hookup the css compiler from www
public/bundle.css: stylesheets/index.css $(css_files) node_modules
	cp $< $@

# Each profile (glob file with file patterns to include) from
# `bundles/<profile>.bundle` is applied to each example folder of the
# form `bundles/<example>/`.
example_profiles := $(shell find bundles -maxdepth 1 -name "*.bundle")
example_code_bundle_dirs := $(shell find bundles -maxdepth 1 -mindepth 1 -type d)
example_files := $(shell find bundles -mindepth 1)
bundle_temp_file := public/bundle.json.tmp

# Builds the playground bundles for each folder and profile combination.
# This task depends on example_profiles and example_files because we want
# to re-bundle if any of those change. However, the bundle tool works on
# directories, so we pass in example_code_bundle_dirs as the path argument.
public/bundles: $(example_profiles) $(example_files) | node_modules
	$(RM) -rf $@
	mkdir -p $(dir $(bundle_temp_file))
	@for profile in $(example_profiles); do \
			echo "Bundling with profile \"$${profile}\""; \
				for folder in $(example_code_bundle_dirs); do \
				pgbundle --verbose "$${profile}" "$${folder}" > $(bundle_temp_file); \
				mv -f $(bundle_temp_file) "$${folder}/bundle_$$(basename $${profile} .bundle).json"; \
			done; \
		done
	cp -r bundles $(@D)

node_modules: package.json
	@npm prune
	# Temporary workaround: install pgbundle directly from source.
	@npm install $(V23_ROOT)/release/projects/playground/pgbundle
	@npm install
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
	@npm cache clean
	@$(RM) -rf public/bundles
	@$(RM) -rf $(shell find bundles -name "bundle*.json")

.PHONY: lint
lint:
	@jshint .

.PHONY: test
test:
	v23 run ./test.sh
