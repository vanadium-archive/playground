// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Bundle commands support bundling playground examples into JSON objects
// compatible with the playground client. Glob files allow specifying file
// subsets for different implementations of the same example. Bundles specified
// in a configuration file can be loaded into the database as named, default
// examples.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"v.io/x/lib/cmdline"
	"v.io/x/lib/dbutil"
	"v.io/x/playground/lib"
	"v.io/x/playground/lib/bundle"
	"v.io/x/playground/lib/storage"
)

var cmdBundle = &cmdline.Command{
	Name:  "bundle",
	Short: "Default bundle management",
	Long: `
Commands for bundling playground examples and loading default bundles into the
database.
`,
	Children: []*cmdline.Command{cmdBundleMake, cmdBundleBootstrap},
}

var cmdBundleMake = &cmdline.Command{
	Runner: cmdline.RunnerFunc(runBundleMake),
	Name:   "make",
	Short:  "Make a single manually specified bundle",
	Long: `
Bundles the example specified by <root_path>, as filtered by <glob_file>, into
a JSON object compatible with the playground client.
`,
	ArgsName: "<glob_file> <root_path>",
	ArgsLong: bundle.BundleUsage,
}

// TODO(ivanpi): Make a single bundle from config file instead of manually specified.
// TODO(ivanpi): Add bundle metadata (title, description) via config file.
// TODO(ivanpi): Iterate over config file, applying commands to bundles (similar to POSIX find)?
var cmdBundleBootstrap = &cmdline.Command{
	Runner: runWithStorage(runBundleBootstrap),
	Name:   "bootstrap",
	Short:  "Bootstrap bundles from config file into database",
	Long: `
Bundles all examples specified in the bundle config file and saves them as
named default bundles into the database specified by sqlconf, replacing any
existing default examples. Bundle slugs are '<example_name>-<glob_name>'.
`,
}

const (
	defaultBundleCfg = "${V23_ROOT}/release/projects/playground/go/src/v.io/x/playground/bundles/config.json"
)

var (
	flagBundleCfgFile string
	flagBundleDir     string
)

func init() {
	cmdBundleBootstrap.Flags.StringVar(&flagBundleCfgFile, "bundleconf", defaultBundleCfg, "Path to bundle config file. "+bundle.BundleConfigFileDescription)
	cmdBundleBootstrap.Flags.StringVar(&flagBundleDir, "bundledir", "", "Path relative to which paths in the bundle config file are interpreted. If empty, defaults to the config file directory.")
}

// Bundles an example from the specified folder using the specified glob file.
// TODO(ivanpi): Expose --verbose and --empty options.
func runBundleMake(env *cmdline.Env, args []string) error {
	if len(args) != 2 {
		return env.UsageErrorf("exactly two arguments expected")
	}
	bOut, err := bundle.Bundle(env.Stderr, args[0], args[1], true)
	if err != nil {
		return fmt.Errorf("Bundling failed: %v", err)
	}
	fmt.Fprintln(env.Stdout, string(bOut))
	return nil
}

// Returns a cmdline.RunnerFunc for loading all bundles specified in the bundle
// config file into the database as default bundles.
func runBundleBootstrap(env *cmdline.Env, args []string) error {
	bundleDir := os.ExpandEnv(flagBundleDir)
	// If bundleDir is empty, interpret paths relative to bundleCfg directory.
	if bundleDir == "" {
		bundleDir = filepath.Dir(os.ExpandEnv(flagBundleCfgFile))
	}
	bundleCfg, err := bundle.ParseConfigFromFile(os.ExpandEnv(flagBundleCfgFile), bundleDir)
	if err != nil {
		return fmt.Errorf("Failed parsing bundle config from %q: %v", os.ExpandEnv(flagBundleCfgFile), err)
	}

	var newDefBundles []*storage.NewBundle
	for _, example := range bundleCfg.Examples {
		fmt.Fprintf(env.Stdout, "Bundling example: %s (%q)\n", example.Name, example.Path)

		for _, globName := range example.Globs {
			glob, globExists := bundleCfg.Globs[globName]
			if !globExists {
				return fmt.Errorf("Unknown glob %q", globName)
			}
			fmt.Fprintf(env.Stdout, "> glob: %s (%q)\n", globName, glob.Path)

			bOut, err := bundle.Bundle(env.Stderr, glob.Path, example.Path, false)
			if err != nil {
				return fmt.Errorf("Bundling %s with %s failed: %v", example.Name, globName, err)
			}

			// Append the bundle and metadata to new default bundles.
			newDefBundles = append(newDefBundles, &storage.NewBundle{
				BundleDesc: storage.BundleDesc{
					Slug: storage.EmptyNullString(example.Name + "-" + globName),
				},
				Json: string(bOut),
			})
		}
	}

	if *flagDryRun {
		fmt.Fprintf(env.Stdout, "Run without dry run to load %d bundles into database\n", len(newDefBundles))
	} else {
		// Unmark old default bundles and store new ones.
		if err := storage.ReplaceDefaultBundles(newDefBundles); err != nil {
			return fmt.Errorf("Failed to replace default bundles: %v", err)
		}
		fmt.Fprintf(env.Stdout, "Successfully loaded %d bundles into database\n", len(newDefBundles))
	}
	return nil
}

// runWithStorage is a wrapper method that handles opening and closing the
// database connections used by `v.io/x/playground/lib/storage`.
func runWithStorage(fx cmdline.RunnerFunc) cmdline.RunnerFunc {
	return func(env *cmdline.Env, args []string) (rerr error) {
		if *flagSQLConf == "" {
			return env.UsageErrorf("SQL configuration file (-sqlconf) must be provided")
		}

		// Parse SQL configuration file and set up TLS.
		dbConf, err := dbutil.ActivateSqlConfigFromFile(*flagSQLConf)
		if err != nil {
			return fmt.Errorf("Error parsing SQL configuration: %v", err)
		}
		// Connect to storage backend.
		if err := storage.Connect(dbConf); err != nil {
			return fmt.Errorf("Error opening database connection: %v", err)
		}
		// Best effort close.
		defer func() {
			if cerr := storage.Close(); cerr != nil {
				cerr = fmt.Errorf("Failed closing database connection: %v", cerr)
				rerr = lib.MergeErrors(rerr, cerr, "\n")
			}
		}()

		// Run wrapped function.
		return fx(env, args)
	}
}
