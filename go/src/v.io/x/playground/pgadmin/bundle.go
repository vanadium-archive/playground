// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Bundle commands support bundling playground examples into JSON objects
// compatible with the playground client. Glob filters allow specifying file
// subsets for different implementations of the same example. Bundles specified
// in a configuration file can be individually bundled or loaded into the
// database as named, default examples.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"v.io/x/lib/cmdline"
	"v.io/x/lib/dbutil"
	"v.io/x/playground/lib"
	"v.io/x/playground/lib/bundle/bundler"
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
	Short:  "Make a single bundle from config file",
	Long: `
Bundles the example named <example>, as filtered by <glob_spec>, specified
in the bundle config file into a JSON object compatible with the playground
client.
`,
	ArgsName: "<example> <glob_spec>",
	ArgsLong: `
<example>: Name of example in config file to be bundled.

<glob_spec>: Name of glob spec in config file to apply when bundling example.
             Glob spec must be referenced by the example as a valid choice.
`,
}

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
	defaultBundleCfg = "${JIRI_ROOT}/release/projects/playground/go/src/v.io/x/playground/bundles/config.json"
)

var (
	flagBundleCfgFile string
	flagBundleDir     string
	flagEmpty         bool
)

func init() {
	cmdBundle.Flags.StringVar(&flagBundleCfgFile, "bundleconf", defaultBundleCfg, "Path to bundle config file. "+bundler.BundleConfigFileDescription)
	cmdBundle.Flags.StringVar(&flagBundleDir, "bundledir", "", "Path relative to which paths in the bundle config file are interpreted. If empty, defaults to the config file directory.")
	cmdBundle.Flags.BoolVar(&flagEmpty, "empty", false, "Omit file contents in bundle, include only paths and metadata.")
}

// Bundles an example from the specified folder using the specified glob.
func runBundleMake(env *cmdline.Env, args []string) error {
	if len(args) != 2 {
		return env.UsageErrorf("exactly two arguments expected")
	}
	exampleName, globName := args[0], args[1]
	emptyFlagWarn(env)

	bundleCfg, err := parseBundleConfig(env)
	if err != nil {
		return err
	}

	glob, globExists := bundleCfg.Globs[globName]
	if !globExists {
		return fmt.Errorf("Unknown glob: %s", globName)
	}

	for _, example := range bundleCfg.Examples {
		if example.Name == exampleName {
			globValid := false
			for _, gn := range example.Globs {
				if gn == globName {
					globValid = true
				}
			}
			if !globValid {
				return fmt.Errorf("Invalid glob for example %s: %s", example.Name, globName)
			}

			bOut, err := bundler.MakeBundleJson(example.Path, glob.Patterns, flagEmpty)
			if err != nil {
				return fmt.Errorf("Bundling %s with %s failed: %v", example.Name, globName, err)
			}
			fmt.Fprintln(env.Stdout, string(bOut))
			if logVerbose() {
				fmt.Fprintf(env.Stderr, "Bundled %s using %s\n", example.Name, globName)
			}

			return nil
		}
	}
	return fmt.Errorf("Unknown example: %s", exampleName)
}

// Returns a cmdline.RunnerFunc for loading all bundles specified in the bundle
// config file into the database as default bundles.
func runBundleBootstrap(env *cmdline.Env, args []string) error {
	emptyFlagWarn(env)
	bundleCfg, err := parseBundleConfig(env)
	if err != nil {
		return err
	}

	var newDefBundles []*storage.NewBundle
	for _, example := range bundleCfg.Examples {
		if logVerbose() {
			fmt.Fprintf(env.Stderr, "Bundling example: %s (%q)\n", example.Name, example.Path)
		}

		for _, globName := range example.Globs {
			glob, globExists := bundleCfg.Globs[globName]
			if !globExists {
				return fmt.Errorf("Unknown glob: %s", globName)
			}
			if logVerbose() {
				fmt.Fprintf(env.Stderr, "> glob: %s\n", globName)
			}

			bOut, err := bundler.MakeBundleJson(example.Path, glob.Patterns, flagEmpty)
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
		fmt.Fprintf(env.Stderr, "Run without dry run to load %d bundles into database\n", len(newDefBundles))
	} else {
		// Unmark old default bundles and store new ones.
		if err := storage.ReplaceDefaultBundles(newDefBundles); err != nil {
			return fmt.Errorf("Failed to replace default bundles: %v", err)
		}
		if logVerbose() {
			fmt.Fprintf(env.Stderr, "Successfully loaded %d bundles into database\n", len(newDefBundles))
		}
	}
	return nil
}

func emptyFlagWarn(env *cmdline.Env) {
	if logVerbose() && flagEmpty {
		fmt.Fprintf(env.Stderr, "Flag -empty set, omitting file contents\n")
	}
}

func parseBundleConfig(env *cmdline.Env) (*bundler.Config, error) {
	bundleCfgFile := os.ExpandEnv(flagBundleCfgFile)
	bundleDir := os.ExpandEnv(flagBundleDir)
	// If bundleDir is empty, interpret paths relative to bundleCfg directory.
	if bundleDir == "" {
		bundleDir = filepath.Dir(bundleCfgFile)
	}
	bundleCfg, err := bundler.ParseConfigFromFile(bundleCfgFile, bundleDir)
	if err != nil {
		return nil, fmt.Errorf("Failed parsing bundle config from %q: %v", bundleCfgFile, err)
	}
	return bundleCfg, nil
}

// runWithStorage is a wrapper method that handles opening and closing the
// database connections used by `v.io/x/playground/lib/storage`.
func runWithStorage(fx cmdline.RunnerFunc) cmdline.RunnerFunc {
	return func(env *cmdline.Env, args []string) (rerr error) {
		if !*flagDryRun {
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
		}

		// Run wrapped function.
		return fx(env, args)
	}
}
