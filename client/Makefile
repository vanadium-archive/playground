PATH := bin:node_modules/.bin:$(PATH)
PATH := $(PATH):$(VANADIUM_ROOT)/environment/cout/node/bin
SHELL := /bin/bash -euo pipefail
export GOPATH := $(realpath $(shell pwd)/../go):$(GOPATH)
export VDLPATH := $(GOPATH)

.DELETE_ON_ERROR:

js_files := $(shell find src/javascript -name "*.js")
css_files := $(shell find src/stylesheets -name "*.css")
static_files := $(shell find src/static -name "*")

.DEFAULT_GOAL := public

.PHONY: deploy-staging
deploy-staging: public
	git rev-parse --verify HEAD >> public/version
	gcloud config set account vanadium-staging
	gsutil -m rsync -d -r public gs://staging.playground.v.io

public: build/playground.js $(css_files) $(static_files) src/example_bundles
	$(RM) -rf $@ build/$@
	mkdir -p build/$@
	cp -r build/playground.js* build/$@
	cp -r src/stylesheets build/$@
	cp -r src/static/. build/$@
	cp -r bundles build/$@
	mv build/$@ .

build/playground.js: src/javascript/index.js $(js_files) node_modules
	mkdir -p $(@D)
	browserify -d $< -p [ minifyify --map $(@F).map --output $@.map ] -o $@

# All paths of the form bundles/<project>/<example>.
example_code_bundle_dirs := $(shell find bundles -maxdepth 2 -mindepth 2)
example_code_files := $(shell find bundles -mindepth 2)

# Builds the playground bundles for the examples.
# This is an empty target.
# See http://www.gnu.org/software/make/manual/html_node/Empty-Targets.html
# This task depends on example_code_files because we want to re-bundle if any of
# those change. However, the bundle tool works on directories, so we pass in
# example_code_bundle_dirs as the argument.
src/example_bundles: $(example_code_files) node_modules
	pgbundle $(example_code_bundle_dirs)
	touch $@

node_modules: package.json
	@npm prune
	# Temporary workaround: install pgbundle directly from source.
	@cd ../pgbundle && npm link
	npm link pgbundle
	@touch $<
	# End workaround.
	npm install
	@touch $@

.PHONY: start
start: deploy
	npm start

.PHONY: clean
clean:
	@$(RM) -rf public
	@$(RM) -rf build
	@$(RM) -rf node_modules
	@$(RM) -rf npm-debug.log

.PHONY: distclean
distclean: clean
	@npm cache clean
	@$(RM) -rf src/example_bundles
	@$(RM) -rf $(shell find bundles -name "bundle.json")

.PHONY: lint
lint:
	@jshint .

.PHONY: test
test:
	v23 run ./test.sh
