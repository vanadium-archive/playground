#!/bin/bash

# Tests that default playground examples execute successfully.
# If any new examples are added, they should be appended to $EXAMPLES below.

# To debug playground compile errors you can build examples locally, e.g.:
# $ cd bundles/fortune/ex0_go/src
# $ GOPATH=$(dirname $(pwd)) VDLPATH=$(dirname $(pwd)) v23 go install ./...

# v.io/core/shell/lib/shell_test.sh sourced via playground/lib/pg_test_util.sh
# (shell_test.sh has side effects, should not be sourced again)
source "$(go list -f {{.Dir}} playground)/../../../client/lib/shell/pg_test_util.sh"

main() {
  cd "${shell_test_WORK_DIR}"

  setup_environment

  build_go_binaries
  install_vanadium_js
  install_pgbundle

  local -r PG_BUNDLES_DIR="${PLAYGROUND_ROOT}/client/bundles"

  local -r EXAMPLES="fortune/ex0_go fortune/ex0_js"

  for e in $EXAMPLES; do
    local d="${PG_BUNDLES_DIR}/${e}"
    echo -e "\n\n>>>>> Test ${d}\n\n"
    test_pg_example "${d}" "-v=true --runTimeout=5000" || shell_test::fail "${d}: failed to run"
    # TODO(sadovsky): Make this "clean exit" check more robust.
    grep -q "\"Exited cleanly.\"" builder.out || shell_test::fail "${d}: did not exit cleanly"
    rm -f builder.out
  done

  shell_test::pass
}

main "$@"
