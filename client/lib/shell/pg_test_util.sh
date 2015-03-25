#!/bin/bash
# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Utilities for testing the playground builder tool.
# Used by tests in client and go/src/playground.

# PLAYGROUND_ROOT is obtained relative to the playground package in
# ${PLAYGROUND_ROOT}/go/src/playground .
# Assumes the playground package is included in GOPATH.
PLAYGROUND_ROOT="$(CDPATH="" cd -P $(go list -f {{.Dir}} playground)/../../.. && pwd)"

source "$(go list -f {{.Dir}} playground)/../../../client/lib/shell/shell_test.sh"

# Sets up environment variables required for the tests.
setup_environment() {
  export GOPATH="$(pwd):$(v23 env GOPATH)"
  export VDLPATH="$(pwd):$(v23 env VDLPATH)"
  export PATH="$(pwd):${shell_test_BIN_DIR}:${VANADIUM_ROOT}/environment/cout/node/bin:${PATH}"

  # We unset all environment variables that supply a principal in order to
  # simulate production playground setup.
  unset VEYRON_CREDENTIALS
  unset VEYRON_AGENT_FD
}

# Installs the release/javascript/core library and makes it accessible to
# Javascript files in the Vanadium playground test under the module name
# 'vanadium'.
install_vanadium_js() {
  # TODO(nlacasse): Once release/javascript/core is publicly available in npm, replace this
  # with "npm install vanadium".
  npm install --production "${VANADIUM_ROOT}/release/javascript/core"
}

# Installs the pgbundle tool.
install_pgbundle() {
  npm install --production "${PLAYGROUND_ROOT}/pgbundle"
}

# Installs various go binaries.
build_go_binaries() {
  shell_test::build_go_binary 'v.io/x/ref/cmd/principal'
  shell_test::build_go_binary 'v.io/x/ref/services/proxy/proxyd'
  shell_test::build_go_binary 'v.io/x/ref/services/mounttable/mounttabled'
  shell_test::build_go_binary 'v.io/x/ref/services/wsprd'
  shell_test::build_go_binary 'v.io/x/ref/cmd/vdl'
  shell_test::build_go_binary 'playground/builder'
}

# Bundles a playground example and tests it using builder.
# $1: root directory of example to test
# $2: glob file with file patterns to bundle from $1
# $3: arguments to call builder with
test_pg_example() {
  local -r PGBUNDLE_DIR="$1"
  local -r PATTERN_FILE="$2"
  local -r BUILDER_ARGS="$3"

  # Create a fresh dir to save the bundle and run builder in.
  local -r TEMP_DIR=$(shell::tmp_dir)

  ./node_modules/.bin/pgbundle --verbose "${PATTERN_FILE}" "${PGBUNDLE_DIR}" > "${TEMP_DIR}/test.json" || return

  local -r ORIG_DIR=$(pwd)
  pushd "${TEMP_DIR}"

  ln -s "${ORIG_DIR}/node_modules" ./  # for release/javascript/core
  "${shell_test_BIN_DIR}/builder" ${BUILDER_ARGS} < "test.json" 2>&1 | tee builder.out
  # Move builder output to original dir for verification.
  mv builder.out "${ORIG_DIR}"

  popd
}
