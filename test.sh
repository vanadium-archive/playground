#!/bin/bash

# Tests the playground builder tool.

# TODO(sadovsky): Much of the setup code below also exists in
# veyron-www/test/playground_test.sh.

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

# Sets up a directory with the given files, then runs builder.
test_with_files() {
  local -r TESTDATA_DIR="$(shell::go_package_dir veyron.io/playground/testdata)"

  # Write input files to a fresh dir, then run pgbundle.
  local -r PGBUNDLE_DIR=$(shell::tmp_dir)
  for f in $@; do
    fdir="${PGBUNDLE_DIR}/$(dirname ${f})"
    mkdir -p "${fdir}"
    cp "${TESTDATA_DIR}/${f}" "${fdir}/"
  done

  ./node_modules/.bin/pgbundle "${PGBUNDLE_DIR}"

  # Create a fresh dir to run bundler from.
  local -r ORIG_DIR=$(pwd)
  pushd $(shell::tmp_dir)
  ln -s "${ORIG_DIR}/node_modules" ./  # for veyron.js
  "${shell_test_BIN_DIR}/builder" -v=0 --includeVeyronEnv=true < "${PGBUNDLE_DIR}/bundle.json" 2>&1 | tee builder.out
  # Move builder output to original dir for verification.
  mv builder.out "${ORIG_DIR}"
  popd
}

main() {
  cd "${shell_test_WORK_DIR}"

  export GOPATH="$(pwd):$(veyron env GOPATH)"
  export VDLPATH="$(pwd):$(veyron env VDLPATH)"
  export PATH="$(pwd):${shell_test_BIN_DIR}:${VEYRON_ROOT}/environment/cout/node/bin:${PATH}"
  unset VEYRON_CREDENTIALS

  build_go_binaries
  install_veyron_js
  install_pgbundle

  echo -e "\n\n>>>>> Test as the same principal\n\n"

  test_with_files "src/pingpong/wire.vdl" "src/pong/pong.go" "src/ping/ping.go" || shell_test::fail "line ${LINENO}: basic ping (go -> go)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  test_with_files "src/pong/pong.js" "src/ping/ping.js" || shell_test::fail "line ${LINENO}: basic ping (js -> js)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  test_with_files "src/pong/pong.go" "src/ping/ping.js" "src/pingpong/wire.vdl" || shell_test::fail "line ${LINENO}: basic ping (js -> go)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  test_with_files "src/pong/pong.js" "src/ping/ping.go" "src/pingpong/wire.vdl" || shell_test::fail "line ${LINENO}: basic ping (go -> js)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  echo -e "\n\n>>>>> Test with authorized blessings\n\n"

  test_with_files "src/pong/pong.go" "src/ping/ping.go" "src/pingpong/wire.vdl" "src/ids/authorized.id" || shell_test::fail "line ${LINENO}: authorized id (go -> go)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/ids/authorized.id" || shell_test::fail "line ${LINENO}: authorized id (js -> js)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  echo -e "\n\n>>>>> Test with expired blessings\n\n"

  test_with_files "src/pong/pong.go" "src/ping/ping.go" "src/pingpong/wire.vdl" "src/ids/expired.id" || shell_test::fail  "line ${LINENO}: expired id (go -> go)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with expired id succeeded (go -> go)"

  test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/ids/expired.id" || shell_test::fail  "line ${LINENO}: expired id (js -> js)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with expired id succeeded (js -> js)"

  echo -e "\n\n>>>>> Test with unauthorized blessings\n\n"

  test_with_files "src/pong/pong.go" "src/ping/ping.go" "src/pingpong/wire.vdl" "src/ids/unauthorized.id" || shell_test::fail  "line ${LINENO}: unauthorized id (go -> go)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with unauthorized id succeeded (go -> go)"

  test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/ids/unauthorized.id" || shell_test::fail  "line ${LINENO}: unauthorized id (js -> js)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with unauthorized id succeeded (js -> js)"

  shell_test::pass
}

main "$@"
