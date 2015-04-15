#!/bin/bash
# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Tests the playground builder tool.

# v.io/core/shell/lib/shell_test.sh sourced via
# playground/client/lib/shell/pg_test_util.sh (shell_test.sh has side
# effects, should not be sourced again)
source "${V23_ROOT}/release/projects/playground/client/lib/shell/pg_test_util.sh"

# Sets up a glob file with the given files, then runs builder.
test_with_files() {
  local -r TESTDATA_DIR="${V23_ROOT}/release/projects/playground/go/src/v.io/x/playground/testdata"

  # Write input file paths to the glob file.
  local -r CONFIG_FILE="$(shell::tmp_dir)/test.bundle"
  echo "$*" | tr ' ' '\n' > "${CONFIG_FILE}"

  test_pg_example "${TESTDATA_DIR}" "${CONFIG_FILE}" "-v=true --includeV23Env=true --runTimeout=5s"
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
