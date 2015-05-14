// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements bundling playground example files from a specified directory,
// filtered using a specified glob file, into a JSON object compatible with
// the playground client.
// Currently implemented as a thin wrapper around the `pgbundle` executable
// written in Node.js. Assumes that pgbundle has been installed in PATH or
// by running `make pgbundle` in pgPackageDir.

// TODO(ivanpi): Port pgbundle to Go.

package bundle

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	nodeBinDir   = "${V23_ROOT}/third_party/cout/node/bin"
	pgPackageDir = "${V23_ROOT}/release/projects/playground/go/src/v.io/x/playground"
)

const BundleUsage = `
<glob_file>: Path to file containing a list of glob patterns, one per line.
             The bundle includes only files with path suffixes matching one
             of the globs. Each glob must match at least one file, otherwise
             bundling fails with a non-zero exit code.
<root_path>: Path to directory where files matching glob patterns are taken
             from.
`

// Bundles files from rootPath matching patterns in globFile. See BundleUsage.
// Errors and (if verbose) informational messages are written to errout.
func Bundle(errout io.Writer, globFile, rootPath string, verbose bool) (outBundle []byte, rerr error) {
	nodePath, err := resolveBinary("node", nodeBinDir)
	if err != nil {
		return nil, err
	}
	pgBundlePath, err := resolveBinary("pgbundle", filepath.Join(pgPackageDir, "node_modules", ".bin"))
	if err != nil {
		return nil, err
	}
	verboseFlag := "--no-verbose"
	if verbose {
		verboseFlag = "--verbose"
	}
	cmdPGBundle := exec.Command(nodePath, pgBundlePath, verboseFlag, globFile, rootPath)
	cmdPGBundle.Dir = os.ExpandEnv(pgPackageDir)
	cmdPGBundle.Stderr = errout
	return cmdPGBundle.Output()
}

// Searches for a binary with the given name in PATH. If not found, defaults to
// the binary with the same name in defaultDir.
func resolveBinary(binName, defaultDir string) (string, error) {
	var errPath, errDef error
	binPath, errPath := exec.LookPath(binName)
	if errPath != nil {
		binPath, errDef = exec.LookPath(filepath.Join(os.ExpandEnv(defaultDir), binName))
	}
	if errDef != nil {
		return "", fmt.Errorf("failed to resolve binary %q: %v; %v", binName, errPath, errDef)
	}
	return binPath, nil
}
