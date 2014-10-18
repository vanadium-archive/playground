#!/bin/bash

# Tests the playgrounds embedded in the website.

# TODO(sadovsky): Much of the setup code below also exists in
# veyron.io/veyron/veyron/tools/playground/test.sh.

source "${VEYRON_ROOT}/scripts/lib/shell_test.sh"

# Installs the veyron.js library and makes it accessible to javascript files in
# the veyron playground test folder under the module name 'veyron'.
install_veyron_js() {
  # TODO(nlacasse): Once veyron.js is publicly available in npm, replace this
  # with "npm install veyron".
  pushd "${VEYRON_ROOT}/veyron.js"
  npm link
  popd
  npm link veyron
}

# Installs the pgbundle tool.
install_pgbundle() {
  pushd "${VEYRON_ROOT}/veyron/go/src/veyron.io/veyron/veyron/tools/playground/pgbundle"
  npm link
  popd
  npm link pgbundle
}

# Installs various go binaries.
build_go_binaries() {
  # Note that "go build" puts built binaries in $(pwd), but only if they are
  # built one at a time. So much for the principle of least surprise...
  local -r V="veyron.io/veyron/veyron"
  veyron go build $V/tools/identity || shell_test::fail "line ${LINENO}: failed to build 'identity'"
  veyron go build $V/services/proxy/proxyd || shell_test::fail "line ${LINENO}: failed to build 'proxyd'"
  veyron go build $V/services/mounttable/mounttabled || shell_test::fail "line ${LINENO}: failed to build 'mounttabled'"
  veyron go build $V/tools/playground/builder || shell_test::fail "line ${LINENO}: failed to build 'builder'"
  veyron go build veyron.io/veyron/veyron2/vdl/vdl || shell_test::fail "line ${LINENO}: failed to build 'vdl'"
  veyron go build veyron.io/wspr/veyron/services/wsprd || shell_test::fail "line ${LINENO}: failed to build 'wsprd'"
}

# Tests a single example (i.e. a single embedded playground).
test_example() {
  local -r PGBUNDLE_DIR="$1"
  ./node_modules/.bin/pgbundle "${PGBUNDLE_DIR}"

  # Create a fresh dir to run bundler from.
  local -r ORIG_DIR=$(pwd)
  pushd $(shell::tmp_dir)
  ln -s "${ORIG_DIR}/node_modules" ./  # for veyron.js
  "${ORIG_DIR}/builder" < "${PGBUNDLE_DIR}/bundle.json" 2>&1 > builder.out
  local -r OK=$?
  popd
  [ $OK ] || shell_test::fail "${PGBUNDLE_DIR}"
}

main() {
  local -r WWWDIR="$(pwd)"
  cd $(shell::tmp_dir)

  export GOPATH="$(pwd):$(veyron env GOPATH)"
  export VDLPATH="$(pwd):$(veyron env VDLPATH)"
  export PATH="$(pwd):${VEYRON_ROOT}/environment/cout/node/bin:${PATH}"

  build_go_binaries
  install_veyron_js
  install_pgbundle

  local -r EXAMPLE_DIRS=$(find "${WWWDIR}/content/playgrounds/code" -maxdepth 2 -mindepth 2)
  for d in $EXAMPLE_DIRS; do
    test_example "$d"
  done

  shell_test::pass
}

main "$@"
