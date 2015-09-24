// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"

	"v.io/x/playground/lib/bundle/bundler"
	_ "v.io/x/ref/runtime/factories/generic"
	"v.io/x/ref/test/expect"
	"v.io/x/ref/test/v23tests"
)

//go:generate jiri test generate

var (
	vanadiumRoot, nodejsRoot, playgroundRoot string
)

func initTest(i *v23tests.T) (builder *v23tests.Binary) {
	vanadiumRoot = os.Getenv("JIRI_ROOT")
	if len(vanadiumRoot) == 0 {
		i.Fatal("JIRI_ROOT must be set")
	}
	nodejsRoot = filepath.Join(vanadiumRoot, "third_party", "cout", "node", "bin")

	i.BuildGoPkg("v.io/x/ref/services/wspr/wsprd", "-a", "-tags", "wspr")
	i.BuildGoPkg("v.io/x/ref/cmd/principal")
	i.BuildGoPkg("v.io/x/ref/cmd/vdl")
	i.BuildGoPkg("v.io/x/ref/services/mounttable/mounttabled")
	i.BuildGoPkg("v.io/x/ref/services/proxy/proxyd")

	playgroundRoot = filepath.Join(vanadiumRoot, "release", "projects", "playground")

	npmInstall(i, filepath.Join(vanadiumRoot, "release/javascript/core"))

	return i.BuildGoPkg("v.io/x/playground/builder")
}

func npmInstall(i *v23tests.T, dir string) {
	npmBin := i.BinaryFromPath(filepath.Join(nodejsRoot, "npm"))
	npmBin.Run("install", "--production", dir)
}

// Bundles a playground example and tests it using builder.
// - dir is the root directory of example to test
// - globList is the list of glob patterns specifying files to use from dir
// - args are the arguments to call builder with
func runPGExample(i *v23tests.T, builder *v23tests.Binary, dir string, globList []string, args ...string) *v23tests.Invocation {
	bundle, err := bundler.MakeBundleJson(dir, globList, false)
	if err != nil {
		i.Fatalf("%s: bundler: failed: %v", i.Caller(1), err)
	}

	tmp := i.NewTempDir("")
	cwd := i.Pushd(tmp)
	defer i.Popd()
	old := filepath.Join(cwd, "node_modules")
	if err := os.Symlink(old, filepath.Join(".", filepath.Base(old))); err != nil {
		i.Fatalf("%s: symlink: failed: %v", i.Caller(1), err)
	}

	PATH := "PATH=" + i.BinDir() + ":" + nodejsRoot
	if path := os.Getenv("PATH"); len(path) > 0 {
		PATH += ":" + path
	}
	stdin := bytes.NewBuffer(bundle)
	return builder.WithEnv(PATH).WithStdin(stdin).Start(args...)
}

// Echoes invocation output to stdout/stderr in addition to checking for
// expected patterns.
func expectAndEcho(inv *v23tests.Invocation, patterns ...string) {
	es := expect.NewSession(inv.Environment(), io.TeeReader(inv.Stdout(), os.Stdout), time.Minute)
	es.ExpectSetEventuallyRE(patterns...)
	inv.WaitOrDie(os.Stdout, os.Stderr)
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
			inv := runPGExample(i, builderBin, testdataDir, files, "--verbose=true", "--includeV23Env=true", "--runTimeout=5s")
			i.Logf("test: %s", c.name)
			expectAndEcho(inv, patterns...)
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

// Tests that default playground examples specified in `config.json` execute
// successfully.
func V23TestPlaygroundBundles(i *v23tests.T) {
	i.Pushd(i.NewTempDir(""))
	defer i.Popd()
	builderBin := initTest(i)

	bundlesDir := filepath.Join(playgroundRoot, "go", "src", "v.io", "x", "playground", "bundles")
	bundlesCfgFile := filepath.Join(bundlesDir, "config.json")
	bundlesCfg, err := bundler.ParseConfigFromFile(bundlesCfgFile, bundlesDir)
	if err != nil {
		i.Fatalf("%s: failed parsing bundle config from %q: %v", i.Caller(0), bundlesCfgFile, err)
	}

	for _, example := range bundlesCfg.Examples {
		i.Logf("Test example %s (%q)", example.Name, example.Path)

		for _, globName := range example.Globs {
			glob, globExists := bundlesCfg.Globs[globName]
			if !globExists {
				i.Fatalf("%s: unknown glob %q", i.Caller(0), globName)
			}

			inv := runPGExample(i, builderBin, example.Path, glob.Patterns, "--verbose=true", "--runTimeout=5s")
			i.Logf("glob: %s", globName)
			expectAndEcho(inv, example.Output...)
		}
	}
}
