#!/bin/bash

# Tests that all embedded playgrounds execute successfully.

# To debug playground compile errors you can build examples locally, e.g.:
# $ cd content/playgrounds/code/fortune/ex0-go/src
# $ GOPATH=$(dirname $(pwd)) VDLPATH=$(dirname $(pwd)) v23 go install ./...

source "$(go list -f {{.Dir}} v.io/core/shell/lib)/shell_test.sh"
source "$(go list -f {{.Dir}} v.io/playground)/lib/pg_test_util.sh"

main() {
  local -r WWWDIR="$(pwd)"
  cd "${shell_test_WORK_DIR}"

  setup_environment

  build_go_binaries
  install_vanadium_js
  install_pgbundle

  local -r EXAMPLE_DIRS=$(find "${WWWDIR}/content/playgrounds/code" -maxdepth 2 -mindepth 2)
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
