package playground_test

import (
	"os"
	"path/filepath"

	"v.io/core/veyron/lib/testutil/v23tests"
	_ "v.io/core/veyron/profiles"
)

//go:generate v23 test generate

var (
	vanadiumRoot, playgroundRoot string
)

func init() {
	vanadiumRoot = os.Getenv("VANADIUM_ROOT")
	if len(vanadiumRoot) == 0 {
		panic("VANADIUM_ROOT must be set")
	}
}

func golist(i *v23tests.T, pkg string) string {
	v23 := filepath.Join(vanadiumRoot, "bin/v23")
	return i.Run(v23, "go", "list", "-f", "{{.Dir}}", pkg)
}

func npmLink(i *v23tests.T, dir, pkg string) {
	npmBin := i.BinaryFromPath(filepath.Join(vanadiumRoot, "environment/cout/node/bin/npm"))
	i.Pushd(dir)
	npmBin.Run("link")
	i.Popd()
	npmBin.Run("link", pkg)
}

// Bundles a playground example and tests it using builder.
// - dir is the root directory of example to test
// - args are the arguments to call builder with
func runPGExample(i *v23tests.T, dir string, args ...string) *v23tests.Invocation {
	i.Run("./node_modules/.bin/pgbundle", dir)
	tmp := i.NewTempDir()
	cwd := i.Pushd(tmp)
	old := filepath.Join(cwd, "node_modules")
	if err := os.Symlink(old, filepath.Join(".", filepath.Base(old))); err != nil {
		i.Fatalf("%s: symlink: failed: %v", i.Caller(2), err)
	}
	bundleName := filepath.Join(dir, "bundle.json")

	stdin, err := os.Open(bundleName)
	if err != nil {
		i.Fatalf("%s: open(%s) failed: %v", i.Caller(2), bundleName, err)
	}
	// TODO(ivanpi): move this out so it only gets invoked once even though
	// the binary is cached.
	builderBin := i.BuildGoPkg("playground/builder")

	PATH := "PATH=" + i.BinDir()
	if path := os.Getenv("PATH"); len(path) > 0 {
		PATH += ":" + path
	}
	defer i.Popd()
	return builderBin.WithEnv(PATH).WithStdin(stdin).Start(args...)
}

// Sets up a directory with the given files, then runs builder.
func testWithFiles(i *v23tests.T, pgRoot string, files ...string) *v23tests.Invocation {
	testdataDir := filepath.Join(pgRoot, "testdata")
	pgBundleDir := i.NewTempDir()
	for _, f := range files {
		fdir := filepath.Join(pgBundleDir, filepath.Dir(f))
		if err := os.MkdirAll(fdir, 0755); err != nil {
			i.Fatalf("%s: mkdir(%q): failed: %v", i.Caller(1), fdir, err)
		}
		i.Run("/bin/cp", filepath.Join(testdataDir, f), fdir)
	}
	return runPGExample(i, pgBundleDir, "-v=true", "--includeV23Env=true", "--runTimeout=5000")
}

func V23TestPlayground(i *v23tests.T) {
	v23tests.RunRootMT(i, "--veyron.tcp.address=127.0.0.1:0")

	i.BuildGoPkg("v.io/core/veyron/tools/principal")
	i.BuildGoPkg("v.io/core/veyron/tools/vdl")
	i.BuildGoPkg("v.io/core/veyron/services/proxy/proxyd")
	i.BuildGoPkg("v.io/core/veyron/services/wsprd")

	playgroundPkg := golist(i, "playground")
	// strip last three directory components, much easier to read in
	// errors than <path>/../../..
	playgroundRoot = filepath.Dir(playgroundPkg)
	playgroundRoot = filepath.Dir(playgroundRoot)
	playgroundRoot = filepath.Dir(playgroundRoot)

	npmLink(i, filepath.Join(vanadiumRoot, "release/javascript/core"), "veyron")
	npmLink(i, filepath.Join(playgroundRoot, "pgbundle"), "pgbundle")

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
