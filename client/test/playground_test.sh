#!/bin/bash

# Tests the playgrounds embedded in the website.

source "${VEYRON_ROOT}/scripts/lib/shell_test.sh"

# Installs the veyron.js library and makes it accessible to javascript files in
# the veyron playground test folder under the module name 'veyron'.
install_veyron_js() {
  # TODO(nlacasse): Once veyron.js is publicly available in npm, replace this
  # with "npm install veyron".
  pushd "${VEYRON_ROOT}/veyron.js"
  "${VEYRON_ROOT}/environment/cout/node/bin/npm" link
  popd
  "${VEYRON_ROOT}/environment/cout/node/bin/npm" link veyron
}

# Installs the pgbundle tool.
install_pgbundle() {
  pushd "${VEYRON_ROOT}/veyron/go/src/veyron.io/veyron/veyron/tools/playground/pgbundle"
  "${VEYRON_ROOT}/environment/cout/node/bin/npm" link
  popd
  "${VEYRON_ROOT}/environment/cout/node/bin/npm" link pgbundle
}

# TODO(sadovsky): Copied from veyron.io/veyron/veyron/tools/playground/test.sh.
# Ideally we could share the playground test setup.
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

test_example() {


  local -r DIR="$1"
  ./node_modules/.bin/pgbundle "$DIR"
  rm -f builder.out
  GOPATH="${DIR}:${GOPATH}"./builder < "${DIR}/bundle.json" 2>&1 > builder.out || shell_test::fail "$DIR"
}

main() {
  local -r WWWDIR="$(pwd)"
  cd $(shell::tmp_dir)

  export GOPATH="$(pwd):$(veyron env GOPATH)"
  export VDLPATH="$(pwd):$(veyron env VDLPATH)"
  export PATH="$(pwd):${PATH}"

  build_go_binaries
  install_veyron_js
  install_pgbundle

  local -r EXAMPLE_DIRS=$(find "${WWWDIR}/content/playgrounds/code" -maxdepth 2 -mindepth 2)
  for d in $EXAMPLE_DIRS; do
    test_example "$d"
  done
}

main "$@"
