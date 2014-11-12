#!/bin/bash

# Tests that all embedded playgrounds execute successfully.

# To build a playground example yourself, do something like:
# $ cd content/playgrounds/code/fortune/ex0-go/src
# $ GOPATH=$(dirname $(pwd)) VDLPATH=$(dirname $(pwd)) veyron go install ./...

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
  pushd "${VEYRON_ROOT}/veyron/javascript/pgbundle"
  npm link
  popd
  npm link pgbundle
}

# Installs various go binaries.
build_go_binaries() {
  shell_test::build_go_binary 'veyron.io/veyron/veyron/tools/principal'
  shell_test::build_go_binary 'veyron.io/veyron/veyron/services/proxy/proxyd'
  shell_test::build_go_binary 'veyron.io/veyron/veyron/services/mounttable/mounttabled'
  shell_test::build_go_binary 'veyron.io/playground/builder'
  shell_test::build_go_binary 'veyron.io/veyron/veyron2/vdl/vdl'
  shell_test::build_go_binary 'veyron.io/wspr/veyron/services/wsprd'
}

# Tests a single example (i.e. a single embedded playground).
test_example() {
  local -r PGBUNDLE_DIR="$1"
  ./node_modules/.bin/pgbundle "${PGBUNDLE_DIR}"

  echo -e "\n\n>>>>> Test ${PGBUNDLE_DIR}\n\n"

  # Create a fresh dir to run bundler from.
  local -r ORIG_DIR=$(pwd)
  pushd $(shell::tmp_dir)
  ln -s "${ORIG_DIR}/node_modules" ./  # for veyron.js
  "${shell_test_BIN_DIR}/builder" < "${PGBUNDLE_DIR}/bundle.json" 2>&1 | tee builder.out
  # TODO(sadovsky): Make this "clean exit" check more robust.
  grep -q "\"Exited cleanly.\"" builder.out || shell_test::fail "${PGBUNDLE_DIR}: did not exit cleanly"
  popd
}

main() {
  local -r WWWDIR="$(pwd)"
  cd "${shell_test_WORK_DIR}"

  export GOPATH="$(pwd):$(veyron env GOPATH)"
  export VDLPATH="$(pwd):$(veyron env VDLPATH)"
  export PATH="$(pwd):${shell_test_BIN_DIR}:${VEYRON_ROOT}/environment/cout/node/bin:${PATH}"
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
