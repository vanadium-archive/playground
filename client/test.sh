#!/bin/bash

# Tests that all embedded playgrounds execute successfully.

# To debug playground compile errors you can build examples locally, e.g.:
# $ cd bundles/fortune/ex0-go/src
# $ GOPATH=$(dirname $(pwd)) VDLPATH=$(dirname $(pwd)) v23 go install ./...

# v.io/core/shell/lib/shell_test.sh sourced via v.io/playground/lib/pg_test_util.sh
# (shell_test.sh has side effects, should not be sourced again)
source "$(go list -f {{.Dir}} v.io/playground)/lib/pg_test_util.sh"

main() {
  cd "${shell_test_WORK_DIR}"

  setup_environment

  build_go_binaries
  install_vanadium_js
  install_pgbundle

  local -r PG_CLIENT_DIR="$(go list -f {{.Dir}} v.io/playground)/client"
  local -r EXAMPLE_DIRS=$(find "${PG_CLIENT_DIR}/bundles" -maxdepth 2 -mindepth 2)
  [ -n "${EXAMPLE_DIRS}" ] || shell_test::fail "no playground examples found"

  for d in $EXAMPLE_DIRS; do
    echo -e "\n\n>>>>> Test ${d}\n\n"
    test_pg_example "${d}" "-v=false" || shell_test::fail "${d}: failed to run"
    # TODO(sadovsky): Make this "clean exit" check more robust.
    grep -q "\"Exited cleanly.\"" builder.out || shell_test::fail "${d}: did not exit cleanly"
    rm -f builder.out
  done

  shell_test::pass
}

main "$@"
