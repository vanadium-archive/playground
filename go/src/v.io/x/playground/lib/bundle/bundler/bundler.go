// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implements bundling playground example files from a specified directory,
// filtered using a specified glob list, into a JSON object compatible with
// the playground client.

package bundler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"v.io/x/playground/lib/bundle"

	"github.com/bmatcuk/doublestar"
)

// Bundles files using MakeBundle and returns the JSON serialized bindle.
func MakeBundleJson(rootPath string, globList []string, empty bool) ([]byte, error) {
	bundle, err := MakeBundle(rootPath, globList, empty)
	if err != nil {
		return nil, err
	}
	return json.Marshal(bundle)
}

// Bundles files in rootPath, filtered by globList. If empty is set, omits file
// contents (includes only paths and metadata).
func MakeBundle(rootPath string, globList []string, empty bool) (*bundle.Bundle, error) {
	rootPath = filepath.Clean(rootPath)
	// The root path must exist and be a directory.
	if fi, err := os.Lstat(rootPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("root path %q does not exist", rootPath)
		} else {
			return nil, fmt.Errorf("error checking root path %q: %v", rootPath, err)
		}
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("root path %q is not a directory", rootPath)
	}

	allPaths := make([]string, 0)
	// Recursively list all regular files in rootPath.
	if err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			// Slash is trimmed separately because rootPath may or may not end with
			// a slash.
			allPaths = append(allPaths, strings.TrimPrefix(strings.TrimPrefix(path, rootPath), "/"))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error listing files in %q: %v", rootPath, err)
	}

	matchingPaths := make(map[string]bool)
	unmatchedGlobs := make([]string, 0)
	// Apply each glob to each file. Each glob must match at least one file; each
	// file is included at most once, even if it matches multiple globs.
	for _, glob := range globList {
		matched := false
		// Globs only need to match a suffix of the file path, so a leading '**' is
		// added.
		suffixGlob := "**/" + glob
		for _, path := range allPaths {
			if ok, err := doublestar.Match(suffixGlob, path); err != nil {
				return nil, fmt.Errorf("error applying glob %q: %v", suffixGlob, err)
			} else if ok {
				matched = true
				matchingPaths[path] = true
			}
		}
		if !matched {
			unmatchedGlobs = append(unmatchedGlobs, glob)
		}
	}
	// If any glob matches no files, bundling fails.
	if len(unmatchedGlobs) > 0 {
		return nil, fmt.Errorf("error bundling %q: unmatched patterns %v", rootPath, unmatchedGlobs)
	}

	files := make([]*indexedCodeFile, 0, len(matchingPaths))
	// Extract sorting indices and strip out "// +build ignore".
	for path, _ := range matchingPaths {
		contents, err := ioutil.ReadFile(filepath.Join(rootPath, path))
		if err != nil {
			return nil, fmt.Errorf("error reading file %q: %v", path, err)
		}
		files = append(files, filterAndIndexCodeFile(path, contents, empty))
	}

	sort.Sort(sortByIndexAndName(files))

	// TODO(ivanpi): Add slug, description, etc?
	var res bundle.Bundle

	for _, icf := range files {
		res.Files = append(res.Files, icf.CodeFile)
	}

	return &res, nil
}

type indexedCodeFile struct {
	*bundle.CodeFile
	index int64
}

// Sorts code files first by index, then by name (path).
type sortByIndexAndName []*indexedCodeFile

var _ sort.Interface = (*sortByIndexAndName)(nil)

func (s sortByIndexAndName) Len() int      { return len(s) }
func (s sortByIndexAndName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortByIndexAndName) Less(i, j int) bool {
	if s[i].index == s[j].index {
		return s[i].Name < s[j].Name
	} else {
		return s[i].index < s[j].index
	}
}

// Strips the first encountered "// +build ignore", extracts and strips the
// sort index and strips leading blank lines.
func filterAndIndexCodeFile(name string, contents []byte, empty bool) *indexedCodeFile {
	lines := bytes.Split(contents, []byte("\n"))
	lines = stripBuildIgnore(lines)
	var index int64
	lines, index = getAndStripIndex(lines)
	lines = stripLeadingBlankLines(lines)
	if empty {
		lines = nil
	}
	return &indexedCodeFile{
		CodeFile: &bundle.CodeFile{
			Name: name,
			Body: string(bytes.Join(lines, []byte("\n"))),
		},
		index: index,
	}
}

// Strips the first encountered "// +build ignore" line in the file and
// returns the remaining lines.
func stripBuildIgnore(file [][]byte) [][]byte {
	res := make([][]byte, 0, len(file))
	found := false
	re := regexp.MustCompile(`^//\s*\+build\s+ignore$`)
	for _, line := range file {
		if !found && re.Match(bytes.TrimSpace(line)) {
			found = true
			continue
		}
		res = append(res, line)
	}
	return res
}

// Strips the first encountered "// pg-index=<num>" line in the file and
// returns the index value and remaining lines. Files with no specified
// index or invalid index are given an infinite index.
func getAndStripIndex(file [][]byte) ([][]byte, int64) {
	res := make([][]byte, 0, len(file))
	var index int64 = math.MaxInt64
	found := false
	re := regexp.MustCompile(`^//\s*pg-index=(-?\d+)$`)
	if re.NumSubexp() != 1 {
		panic("cannot happen: regexp has <> 1 subexp")
	}
	for _, line := range file {
		if !found {
			if match := re.FindSubmatch(bytes.TrimSpace(line)); match != nil {
				if len(match) < 2 {
					panic("cannot happen: missing submatch")
				}
				if parsed, err := strconv.ParseInt(string(match[1]), 10, 64); err == nil {
					found = true
					index = parsed
					continue
				}
				// TODO(ivanpi): Warn otherwise (e.g. index overflow)?
			}
		}
		res = append(res, line)
	}
	return res, index
}

// Strips all blank lines at the beginning of the file.
func stripLeadingBlankLines(file [][]byte) [][]byte {
	res := append([][]byte(nil), file...)
	for len(res) > 0 && len(bytes.TrimSpace(res[0])) == 0 {
		res = res[1:]
	}
	return res
}
