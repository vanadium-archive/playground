// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"v.io/x/playground/lib/bundle/bundler"
	_ "v.io/x/ref/runtime/factories/generic"
	tu "v.io/x/ref/test/testutil"
	"v.io/x/ref/test/v23test"
)

var (
	vanadiumRoot, playgroundRoot string
)

func initTest(t *testing.T, sh *v23test.Shell) (builderPath string) {
	vanadiumRoot = os.Getenv("JIRI_ROOT")
	if len(vanadiumRoot) == 0 {
		t.Fatal("JIRI_ROOT must be set")
	}

	v23test.BuildGoPkg(sh, "v.io/x/ref/cmd/principal")
	v23test.BuildGoPkg(sh, "v.io/x/ref/cmd/vdl")
	v23test.BuildGoPkg(sh, "v.io/x/ref/services/mounttable/mounttabled")
	v23test.BuildGoPkg(sh, "v.io/x/ref/services/xproxy/xproxyd")

	playgroundRoot = filepath.Join(vanadiumRoot, "release", "projects", "playground")

	return v23test.BuildGoPkg(sh, "v.io/x/playground/builder")
}

// Bundles a playground example and tests it using builder.
// - dir is the root directory of example to test
// - globList is the list of glob patterns specifying files to use from dir
// - args are the arguments to call builder with
func runPGExample(t *testing.T, sh *v23test.Shell, builderPath, dir string, globList []string, args ...string) *v23test.Cmd {
	bundle, err := bundler.MakeBundleJson(dir, globList, false)
	if err != nil {
		t.Fatal(tu.FormatLogLine(1, "bundler: failed: %v", err))
	}

	tmp := sh.MakeTempDir()
	sh.Pushd(tmp)
	defer sh.Popd()

	PATH := os.Getenv("V23_BIN_DIR")
	if path := os.Getenv("PATH"); len(path) > 0 {
		PATH += ":" + path
	}
	builder := sh.Cmd(builderPath, args...)
	builder.Vars["PATH"] = PATH
	builder.SetStdinReader(bytes.NewReader(bundle))
	builder.PropagateOutput = true
	builder.Start()
	return builder
}

// Tests the playground builder tool.
func TestV23PlaygroundBuilder(t *testing.T) {
	v23test.SkipUnlessRunningIntegrationTests(t)
	sh := v23test.NewShell(t, nil)
	defer sh.Cleanup()
	sh.Pushd(sh.MakeTempDir())
	defer sh.Popd()
	builderPath := initTest(t, sh)

	cases := []struct {
		name  string
		files []string
	}{
		{"basic ping (go -> go)",
			[]string{"src/pong/pong.go", "src/ping/ping.go", "src/pingpong/wire.vdl"}},
	}

	testdataDir := filepath.Join(playgroundRoot, "go", "src", "v.io", "x", "playground", "testdata")

	runCases := func(authfile string, patterns []string) {
		for _, c := range cases {
			files := c.files
			if len(authfile) > 0 {
				files = append(files, authfile)
			}
			inv := runPGExample(t, sh, builderPath, testdataDir, files, "--verbose=true", "--includeProfileEnv=true", "--runTimeout=5s")
			t.Logf("test: %s", c.name)
			inv.S.ExpectSetEventuallyRE(patterns...)
			inv.Wait()
		}
	}

	t.Logf("Test as the same principal")
	runCases("", []string{"PING", "PONG"})

	t.Logf("Test with authorized blessings")
	runCases("src/ids/authorized.id", []string{"PING", "PONG"})

	t.Logf("Test with expired blessings")
	runCases("src/ids/expired.id", []string{"not authorized"})

	t.Logf("Test with unauthorized blessings")
	runCases("src/ids/unauthorized.id", []string{"not authorized"})
}

// Tests that default playground examples specified in `config.json` execute
// successfully.
func TestV23PlaygroundBundles(t *testing.T) {
	v23test.SkipUnlessRunningIntegrationTests(t)
	sh := v23test.NewShell(t, nil)
	defer sh.Cleanup()
	sh.Pushd(sh.MakeTempDir())
	defer sh.Popd()
	builderPath := initTest(t, sh)

	bundlesDir := filepath.Join(playgroundRoot, "go", "src", "v.io", "x", "playground", "bundles")
	bundlesCfgFile := filepath.Join(bundlesDir, "config.json")
	bundlesCfg, err := bundler.ParseConfigFromFile(bundlesCfgFile, bundlesDir)
	if err != nil {
		t.Fatal(tu.FormatLogLine(0, "failed parsing bundle config from %q: %v", bundlesCfgFile, err))
	}

	for _, example := range bundlesCfg.Examples {
		t.Logf("Test example %s (%q)", example.Name, example.Path)

		for _, globName := range example.Globs {
			glob, globExists := bundlesCfg.Globs[globName]
			if !globExists {
				t.Fatal(tu.FormatLogLine(0, "unknown glob %q", globName))
			}

			inv := runPGExample(t, sh, builderPath, example.Path, glob.Patterns, "--verbose=true", "--runTimeout=5s")
			t.Logf("glob: %s", globName)
			inv.S.ExpectSetEventuallyRE(example.Output...)
			inv.Wait()
		}
	}
}

func TestMain(m *testing.M) {
	v23test.TestMain(m)
}
