// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main_test

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

func initTest(i *v23tests.T) (builder *v23tests.Binary) {
	vanadiumRoot = os.Getenv("V23_ROOT")
	if len(vanadiumRoot) == 0 {
		i.Fatal("V23_ROOT must be set")
	}
	nodejsRoot = filepath.Join(vanadiumRoot, "third_party", "cout", "node", "bin")

	i.BuildGoPkg("v.io/x/ref/services/wspr/wsprd", "-a", "-tags", "wspr")
	i.BuildGoPkg("v.io/x/ref/cmd/principal")
	i.BuildGoPkg("v.io/x/ref/cmd/vdl")
	i.BuildGoPkg("v.io/x/ref/services/mounttable/mounttabled")
	i.BuildGoPkg("v.io/x/ref/services/proxy/proxyd")

	playgroundRoot = filepath.Join(vanadiumRoot, "release", "projects", "playground")

	npmInstall(i, filepath.Join(vanadiumRoot, "release/javascript/core"))
	npmInstall(i, filepath.Join(playgroundRoot, "pgbundle"))

	return i.BuildGoPkg("v.io/x/playground/builder")
}

func npmInstall(i *v23tests.T, dir string) {
	npmBin := i.BinaryFromPath(filepath.Join(nodejsRoot, "npm"))
	npmBin.Run("install", "--production", dir)
}

// Bundles a playground example and tests it using builder.
// - dir is the root directory of example to test
// - globFile is the path to the glob file with file patterns to use from dir
// - args are the arguments to call builder with
func runPGExample(i *v23tests.T, builder *v23tests.Binary, globFile, dir string, args ...string) *v23tests.Invocation {
	nodeBin := i.BinaryFromPath(filepath.Join(nodejsRoot, "node"))
	bundle := nodeBin.Run("./node_modules/.bin/pgbundle", "--verbose", globFile, dir)

	tmp := i.NewTempDir("")
	cwd := i.Pushd(tmp)
	defer i.Popd()
	old := filepath.Join(cwd, "node_modules")
	if err := os.Symlink(old, filepath.Join(".", filepath.Base(old))); err != nil {
		i.Fatalf("%s: symlink: failed: %v", i.Caller(2), err)
	}

	PATH := "PATH=" + i.BinDir() + ":" + nodejsRoot
	if path := os.Getenv("PATH"); len(path) > 0 {
		PATH += ":" + path
	}
	stdin := bytes.NewBufferString(bundle)
	return builder.WithEnv(PATH).WithStdin(stdin).Start(args...)
}

// Sets up a glob file with the given files, then runs builder.
func testWithFiles(i *v23tests.T, builder *v23tests.Binary, testdataDir string, files ...string) *v23tests.Invocation {
	globFile := filepath.Join(i.NewTempDir(""), "test.bundle")
	if err := ioutil.WriteFile(globFile, []byte(strings.Join(files, "\n")+"\n"), 0644); err != nil {
		i.Fatalf("%s: write(%q): failed: %v", i.Caller(1), globFile, err)
	}
	return runPGExample(i, builder, globFile, testdataDir, "-verbose=true", "--includeV23Env=true", "--runTimeout=5s")
}

// Tests the playground builder tool.
func V23TestPlaygroundBuilder(i *v23tests.T) {
	i.Pushd(i.NewTempDir(""))
	defer i.Popd()
	builderBin := initTest(i)

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

	testdataDir := filepath.Join(playgroundRoot, "go", "src", "v.io", "x", "playground", "testdata")

	runCases := func(authfile string, patterns []string) {
		for _, c := range cases {
			files := c.files
			if len(authfile) > 0 {
				files = append(files, authfile)
			}
			inv := testWithFiles(i, builderBin, testdataDir, files...)
			i.Logf("test: %s", c.name)
			inv.ExpectSetEventuallyRE(patterns...)
		}
	}

	i.Logf("Test as the same principal")
	runCases("", []string{"PING", "PONG"})

	i.Logf("Test with authorized blessings")
	runCases("src/ids/authorized.id", []string{"PING", "PONG"})

	i.Logf("Test with expired blessings")
	runCases("src/ids/expired.id", []string{"not authorized"})

	i.Logf("Test with unauthorized blessings")
	runCases("src/ids/unauthorized.id", []string{"not authorized"})
}

// Tests that default playground examples execute successfully.
// If any new examples are added, they should be added to bundles below.
func V23TestPlaygroundBundles(i *v23tests.T) {
	i.Pushd(i.NewTempDir(""))
	defer i.Popd()
	builderBin := initTest(i)

	bundlesDir := filepath.Join(playgroundRoot, "client", "bundles")

	// Add any new examples here.
	bundles := []string{
		"fortune",
	}

	globFiles := []string{
		"go.bundle",
		"js.bundle",
		"go_js.bundle",
		"js_go.bundle",
	}

	for _, bundle := range bundles {
		i.Logf("Test bundle %q", bundle)
		bundlePath := filepath.Join(bundlesDir, bundle)
		for _, globFile := range globFiles {
			globFilePath := filepath.Join(bundlesDir, globFile)
			inv := runPGExample(i, builderBin, globFilePath, bundlePath, "-verbose=true", "--runTimeout=5s")
			i.Logf("glob: %q", globFile)
			// TODO(ivanpi,sadovsky): Make this "clean exit" check more robust.
			inv.ExpectSetEventuallyRE("Exited cleanly\\.")
		}
	}
}
