// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Representation of the example bundle configuration file, parsed from JSON.
// The configuration file specifies combinations of example folders and
// applicable glob files for bundling default examples, as well as expected
// output for verifying their correctness.

package bundle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Description of the bundle configuration file format.
const BundleConfigFileDescription = `File must contain a JSON object of the following form:
   {
    "examples": [ <Example> ... ], (array of Example objects)
    "globs": { "<glob_name>":<Glob> ... } (map of glob names to Glob objects; glob names should be human-readable but URL-friendly)
   }
Example objects have the form:
   {
   	"name": "<name>", (example names should be human-readable but URL-friendly)
   	"path": "<path/to/example/dir>", (path to directory containing files to be filtered by globs and bundled)
   	"globs": [ "<glob_name>" ... ], (names of globs to be applied to the directory; must have corresponding entries in "globs";
   		each example can be bundled into a separate bundle using one of the specified globs)
   	"output": [ "<expected_output_regex>" ... ] (expected output specification for this example, for any applicable glob;
   		each regex must match at least one output line for the test to succeed)
   }
Glob objects have the form:
   {
   	"path": "<path/to/glob_file>" (path to file containing a list of glob patterns;
   		files from the example directory matching at least one pattern will be included in the bundle;
   		each glob pattern must match at least one file for the bundling to succeed)
   }
Non-absolute paths are interpreted relative to a configurable directory, usually the configuration file directory.`

// Parsed bundle configuration. See BundleConfigFileDescription for details.
type Config struct {
	// List of Example folder descriptors.
	Examples []*Example `json:"examples"`
	// Maps glob names to Glob file descriptors.
	Globs map[string]*Glob `json:"globs"`
}

// Represents an example folder. Each specified glob file is applied to the
// folder to produce a separate bundle, representing different implementations
// of the same example.
type Example struct {
	// Human-readable, URL-friendly name.
	Name string `json:"name"`
	// Path to example directory.
	Path string `json:"path"`
	// Names of globs to apply to the directory.
	Globs []string `json:"globs"`
	// Expected output regexes for testing.
	Output []string `json:"output"`
}

// Represents a glob file for filtering bundled files.
type Glob struct {
	// Path to glob file.
	Path string `json:"path"`
}

// Parses configuration from file and normalizes non-absolute paths relative to
// baseDir. Doesn't do consistency verification.
func ParseConfigFromFile(configPath, baseDir string) (*Config, error) {
	cfgJson, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed reading bundle config from %q: %v", configPath, err)
	}
	var cfg Config
	if err := json.Unmarshal(cfgJson, &cfg); err != nil {
		return nil, fmt.Errorf("failed parsing bundle config: %v", err)
	}
	cfg.NormalizePaths(baseDir)
	return &cfg, nil
}

// Canonicalizes example and glob file paths and resolves them relative to
// baseDir.
func (c *Config) NormalizePaths(baseDir string) {
	for _, e := range c.Examples {
		e.Path = normalizePath(e.Path, baseDir)
	}
	for _, g := range c.Globs {
		g.Path = normalizePath(g.Path, baseDir)
	}
}

// If path is not absolute, resolves path relative to baseDir. Otherwise,
// canonicalizes path.
func normalizePath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(baseDir, path)
}
