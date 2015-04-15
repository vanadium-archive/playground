#!/bin/bash
# Copyright 2015 The Vanadium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Tests that default playground examples execute successfully.
# If any new examples are added, they should be appended to $EXAMPLES below.

# To debug playground compile errors you can build examples locally, e.g.:
# $ cd bundles/fortune/src
# $ GOPATH=$(dirname $(pwd)) VDLPATH=$(dirname $(pwd)) v23 go install ./...

# v.io/core/shell/lib/shell_test.sh sourced via
# playground/client/lib/shell/pg_test_util.sh (shell_test.sh has side
# effects, should not be sourced again)
source "${V23_ROOT}/release/projects/playground/client/lib/shell/pg_test_util.sh"

main() {
  cd "${shell_test_WORK_DIR}"

  setup_environment

  build_go_binaries
  install_vanadium_js
  install_pgbundle

  local -r PG_BUNDLES_DIR="${PLAYGROUND_ROOT}/client/bundles"

  local -r EXAMPLES="fortune"

  for e in ${EXAMPLES}; do
    for p in "${PG_BUNDLES_DIR}"/*.bundle; do
      local d="${PG_BUNDLES_DIR}/${e}"
      local description="${e} with $(basename ${p})"
      echo -e "\n\n>>>>> Test ${description}\n\n"
      test_pg_example "${d}" "${p}" "-v=true --runTimeout=5s" || shell_test::fail "${description}: failed to run"
      # TODO(sadovsky): Make this "clean exit" check more robust.
      grep -q "\"Exited cleanly.\"" builder.out || shell_test::fail "${description}: did not exit cleanly"
      rm -f builder.out
    done
  done

  shell_test::pass
}

main "$@"
