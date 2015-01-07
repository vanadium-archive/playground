#!/bin/bash

# Tests the playground builder tool.

# TODO(sadovsky): Much of the setup code below also exists in
# veyron-www/test/playground_test.sh.

source "$(go list -f {{.Dir}} v.io/core/shell/lib)/shell_test.sh"

# Installs the release/javascript/core library and makes it accessible to javascript files in
# the veyron playground test folder under the module name 'veyron'.
install_veyron_js() {
  # TODO(nlacasse): Once release/javascript/core is publicly available in npm, replace this
  # with "npm install veyron".

  pushd "${VANADIUM_ROOT}/release/javascript/vom"
  npm link
  popd
  pushd "${VANADIUM_ROOT}/release/javascript/core"
  npm link vom
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
  shell_test::build_go_binary 'v.io/core/veyron2/vdl/vdl'
  shell_test::build_go_binary 'v.io/playground/builder'
  shell_test::build_go_binary 'v.io/wspr/veyron/services/wsprd'
}

# Sets up a directory with the given files, then runs builder.
test_with_files() {
  local -r TESTDATA_DIR="$(go list -f {{.Dir}} v.io/playground)/testdata"

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
  ln -s "${ORIG_DIR}/node_modules" ./  # for release/javascript/core
  "${shell_test_BIN_DIR}/builder" -v=0 --includeVeyronEnv=true < "${PGBUNDLE_DIR}/bundle.json" 2>&1 | tee builder.out
  # Move builder output to original dir for verification.
  mv builder.out "${ORIG_DIR}"
  popd
}

main() {
  cd "${shell_test_WORK_DIR}"

  export GOPATH="$(pwd):$(v23 env GOPATH)"
  export VDLPATH="$(pwd):$(v23 env VDLPATH)"
  export PATH="$(pwd):${shell_test_BIN_DIR}:${VANADIUM_ROOT}/environment/cout/node/bin:${PATH}"

  # We unset all environment variables that supply a principal in order to
  # simulate production playground setup.
  unset VEYRON_CREDENTIALS
  unset VEYRON_AGENT_FD

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
