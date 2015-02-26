#!/bin/bash

# Tests the playground builder tool.

# v.io/core/shell/lib/shell_test.sh sourced via playground/lib/pg_test_util.sh
# (shell_test.sh has side effects, should not be sourced again)
source "$(go list -f {{.Dir}} playground)/../../../client/lib/shell/pg_test_util.sh"

# Sets up a directory with the given files, then runs builder.
test_with_files() {
  local -r TESTDATA_DIR="$(go list -f {{.Dir}} playground)/testdata"

  # Write input files to a fresh dir before bundling and running them.
  local -r PGBUNDLE_DIR=$(shell::tmp_dir)
  for f in $@; do
    fdir="${PGBUNDLE_DIR}/$(dirname ${f})"
    mkdir -p "${fdir}"
    cp "${TESTDATA_DIR}/${f}" "${fdir}/"
  done

  test_pg_example "${PGBUNDLE_DIR}" "-v=true --includeV23Env=true --runTimeout=5000"
}

main() {
  cd "${shell_test_WORK_DIR}"

  setup_environment

  build_go_binaries
  install_vanadium_js
  install_pgbundle

  echo -e "\n\n>>>>> Test as the same principal\n\n"

  test_with_files "src/pong/pong.go" "src/ping/ping.go" "src/pingpong/wire.vdl" || shell_test::fail "line ${LINENO}: basic ping (go -> go)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/pingpong/wire.vdl" || shell_test::fail "line ${LINENO}: basic ping (js -> js)"
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

  test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/pingpong/wire.vdl" "src/ids/authorized.id" || shell_test::fail "line ${LINENO}: authorized id (js -> js)"
  grep -q PING builder.out || shell_test::fail "line ${LINENO}: no PING"
  grep -q PONG builder.out || shell_test::fail "line ${LINENO}: no PONG"

  echo -e "\n\n>>>>> Test with expired blessings\n\n"

  test_with_files "src/pong/pong.go" "src/ping/ping.go" "src/pingpong/wire.vdl" "src/ids/expired.id" || shell_test::fail  "line ${LINENO}: expired id (go -> go)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with expired id succeeded (go -> go)"

# TODO(bprosnitz) Re-enable with issue #986 (once javascript supports expired blessings).
# test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/pingpong/wire.vdl" "src/ids/expired.id" || shell_test::fail  "line ${LINENO}: expired id (js -> js)"
#  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with expired id succeeded (js -> js)"

  echo -e "\n\n>>>>> Test with unauthorized blessings\n\n"

  test_with_files "src/pong/pong.go" "src/ping/ping.go" "src/pingpong/wire.vdl" "src/ids/unauthorized.id" || shell_test::fail  "line ${LINENO}: unauthorized id (go -> go)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with unauthorized id succeeded (go -> go)"

  test_with_files "src/pong/pong.js" "src/ping/ping.js" "src/pingpong/wire.vdl" "src/ids/unauthorized.id" || shell_test::fail  "line ${LINENO}: unauthorized id (js -> js)"
  grep -q "not authorized" builder.out || shell_test::fail "line ${LINENO}: rpc with unauthorized id succeeded (js -> js)"

  shell_test::pass
}

main "$@"
