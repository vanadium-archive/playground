// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package playground_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	_ "v.io/x/ref/profiles"
	"v.io/x/ref/test/v23tests"
)

//go:generate v23 test generate

var (
	vanadiumRoot, nodejsRoot, playgroundRoot string
)

func init() {
	vanadiumRoot = os.Getenv("V23_ROOT")
	if len(vanadiumRoot) == 0 {
		panic("V23_ROOT must be set")
	}
	nodejsRoot = filepath.Join(vanadiumRoot, "third_party", "cout", "node", "bin")
}

func golist(i *v23tests.T, pkg string) string {
	v23 := filepath.Join(vanadiumRoot, "bin/v23")
	return i.Run(v23, "go", "list", "-f", "{{.Dir}}", pkg)
}

func npmInstall(i *v23tests.T, dir string) {
	npmBin := i.BinaryFromPath(filepath.Join(nodejsRoot, "npm"))
	npmBin.Run("install", "--production", dir)
}

// Bundles a playground example and tests it using builder.
// - dir is the root directory of example to test
// - globFile is the path to the glob file with file patterns to use from dir
// - args are the arguments to call builder with
func runPGExample(i *v23tests.T, globFile, dir string, args ...string) *v23tests.Invocation {
	bundle := i.Run("./node_modules/.bin/pgbundle", "--verbose", globFile, dir)

	tmp := i.NewTempDir()
	cwd := i.Pushd(tmp)
	defer i.Popd()
	old := filepath.Join(cwd, "node_modules")
	if err := os.Symlink(old, filepath.Join(".", filepath.Base(old))); err != nil {
		i.Fatalf("%s: symlink: failed: %v", i.Caller(2), err)
	}

	// TODO(ivanpi): move this out so it only gets invoked once even though
	// the binary is cached.
	builderBin := i.BuildGoPkg("v.io/x/playground/builder")

	PATH := "PATH=" + i.BinDir() + ":" + nodejsRoot
	if path := os.Getenv("PATH"); len(path) > 0 {
		PATH += ":" + path
	}
	stdin := bytes.NewBufferString(bundle)
	return builderBin.WithEnv(PATH).WithStdin(stdin).Start(args...)
}

// Sets up a glob file with the given files, then runs builder.
func testWithFiles(i *v23tests.T, pgRoot string, files ...string) *v23tests.Invocation {
	testdataDir := filepath.Join(pgRoot, "testdata")
	globFile := filepath.Join(i.NewTempDir(), "test.bundle")
	if err := ioutil.WriteFile(globFile, []byte(strings.Join(files, "\n")+"\n"), 0644); err != nil {
		i.Fatalf("%s: write(%q): failed: %v", i.Caller(1), globFile, err)
	}
	return runPGExample(i, globFile, testdataDir, "-v=true", "--includeV23Env=true", "--runTimeout=5s")
}

func V23TestPlayground(i *v23tests.T) {
	i.Pushd(i.NewTempDir())
	defer i.Popd()

	v23tests.RunRootMT(i, "--v23.tcp.address=127.0.0.1:0")

	i.BuildGoPkg("v.io/x/ref/services/wspr/wsprd", "-a", "-tags", "wspr")
	i.BuildGoPkg("v.io/x/ref/cmd/principal")
	i.BuildGoPkg("v.io/x/ref/cmd/vdl")
	i.BuildGoPkg("v.io/x/ref/services/proxy/proxyd")

	playgroundPkg := golist(i, "v.io/x/playground")
	// strip last three directory components, much easier to read in
	// errors than <path>/../../..
	playgroundRoot = filepath.Dir(playgroundPkg)
	playgroundRoot = filepath.Dir(playgroundRoot)
	playgroundRoot = filepath.Dir(playgroundRoot)

	npmInstall(i, filepath.Join(vanadiumRoot, "release/javascript/core"))
	npmInstall(i, filepath.Join(playgroundRoot, "pgbundle"))

	cases := []struct {
		name  string
		files []string
	}{
		{"basic ping (go -> go)",
			[]string{"src/pong/pong.go", "src/ping/ping.go", "src/pingpong/wire.vdl"}},
		{"basic ping (js -> js)",
			[]string{"src/pong/pong.js", "src/ping/ping.js", "src/pingpong/wire.vdl"}},
		{"basic ping (js -> go)",
			[]string{"src/pong/pong.go", "src/ping/ping.js", "src/pingpong/wire.vdl"}},
		{"basic ping (go -> js)",
			[]string{"src/pong/pong.js", "src/ping/ping.go", "src/pingpong/wire.vdl"}},
	}

	runCases := func(authfile string, patterns []string) {
		for _, c := range cases {
			files := c.files
			if len(authfile) > 0 {
				files = append(files, authfile)
			}
			inv := testWithFiles(i, playgroundPkg, files...)
			i.Logf("test: %s", c.name)
			inv.ExpectSetEventuallyRE(patterns...)
		}
	}

	i.Logf("Test as the same principal")
	runCases("", []string{"PING", "PONG"})

	i.Logf("Test with authorized blessings")
	runCases("src/ids/authorized.id", []string{"PING", "PONG"})

	// TODO(bprosnitz) Re-enable with issue #986 (once javascript supports expired blessings).
	//i.Logf("Test with expired blessings")
	//runCases("src/ids/expired.id", []string{"not authorized"})

	i.Logf("Test with unauthorized blessings")
	runCases("src/ids/unauthorized.id", []string{"not authorized"})

}
