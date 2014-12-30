#!/bin/bash

# Tests that all embedded playgrounds execute successfully.

# To debug playground compile errors you can build examples locally, e.g.:
# $ cd content/playgrounds/code/fortune/ex0-go/src
# $ GOPATH=$(dirname $(pwd)) VDLPATH=$(dirname $(pwd)) v23 go install ./...

# TODO(sadovsky): Much of the setup code below also exists in
# v.io/playground/test.sh.

source "$(go list -f {{.Dir}} v.io/core/shell/lib)/shell_test.sh"

# Installs the release/javascript/core library and makes it accessible to javascript files in
# the veyron playground test folder under the module name 'veyron'.
install_veyron_js() {
  # TODO(nlacasse): Once release/javascript/core is publicly available in npm, replace this
  # with "npm install veyron".
  pushd "${VANADIUM_ROOT}/release/javascript/core"
  npm link
  popd
  npm link veyron
}

# Installs the pgbundle tool.
install_pgbundle() {
  pushd "${VANADIUM_ROOT}/release/javascript/pgbundle"
  npm link
  popd
  npm link pgbundle
}

# Installs various go binaries.
build_go_binaries() {
  shell_test::build_go_binary 'v.io/core/veyron/tools/principal'
  shell_test::build_go_binary 'v.io/core/veyron/services/proxy/proxyd'
  shell_test::build_go_binary 'v.io/core/veyron/services/mounttable/mounttabled'
  shell_test::build_go_binary 'v.io/playground/builder'
  shell_test::build_go_binary 'v.io/core/veyron2/vdl/vdl'
  shell_test::build_go_binary 'v.io/wspr/veyron/services/wsprd'
}

# Tests a single example (i.e. a single embedded playground).
test_example() {
  local -r PGBUNDLE_DIR="$1"
  ./node_modules/.bin/pgbundle "${PGBUNDLE_DIR}"

  echo -e "\n\n>>>>> Test ${PGBUNDLE_DIR}\n\n"

  # Create a fresh dir to run bundler from.
  local -r ORIG_DIR=$(pwd)
  pushd $(shell::tmp_dir)
  ln -s "${ORIG_DIR}/node_modules" ./  # for release/javascript/core
  "${shell_test_BIN_DIR}/builder" < "${PGBUNDLE_DIR}/bundle.json" 2>&1 | tee builder.out
  # TODO(sadovsky): Make this "clean exit" check more robust.
  grep -q "\"Exited cleanly.\"" builder.out || shell_test::fail "${PGBUNDLE_DIR}: did not exit cleanly"
  popd
}

main() {
  local -r WWWDIR="$(pwd)"
  cd "${shell_test_WORK_DIR}"

  export GOPATH="$(pwd):$(v23 env GOPATH)"
  export VDLPATH="$(pwd):$(v23 env VDLPATH)"
  export PATH="$(pwd):${shell_test_BIN_DIR}:${VANADIUM_ROOT}/environment/cout/node/bin:${PATH}"
  unset VEYRON_CREDENTIALS

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
