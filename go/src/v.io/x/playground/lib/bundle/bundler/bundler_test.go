// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bundler

import (
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"v.io/x/playground/lib/bundle"
)

type testFile struct {
	// File contents, line by line. Lines must start with '+++' if they should
	// remain in the bundled file, or '---' if they should be filtered out.
	contents []string
	// Sort index expected to be parsed from the file.
	index int64
}

var testFiles = map[string]*testFile{
	"alpha/one/A-5.a": {
		[]string{
			"---",
			"---// +build ignore",
			"---",
			"---\t",
			"+++lalala",
			"+++\t",
		},
		math.MaxInt64,
	},
	"alpha/one/B-3.b": {
		[]string{
			"---// pg-index=12",
			"---",
			"+++hello, world",
		},
		12,
	},
	"alpha/one/foo/C-1.a": {
		[]string{
			"+++//  pg-index=0x32",
			"---//\tpg-index=-123 ",
			"+++// pg-index=222",
			"+++// +build arm",
			"+++",
			"---  // +build\tignore",
			"+++// +build ignore ",
		},
		-123,
	},
	"alpha/two/D-6.d": {
		[]string{
			"+++// pg-index=-12345678123456781234",
			"+++foo",
			"+++",
		},
		math.MaxInt64,
	},
	"beta/E-2.b": {
		[]string{
			"--- ",
			"+++header",
			"---\t// pg-index=1",
			"+++ ",
			"+++foobar",
			"+++",
		},
		1,
	},
	"beta/one/F-7.a": {
		[]string{
			"---",
			"--- ",
			"---",
		},
		math.MaxInt64,
	},
	"beta/two/G-4": {
		[]string{
			"+++Elbereth",
			"---// +build ignore ",
			"+++",
			"--- // pg-index=42",
		},
		42,
	},
}

// Strips '+++' and '---' prefixes from file contents. If filter is set, omits
// lines starting with '---'.
func (tf *testFile) getContents(t *testing.T, filter bool) string {
	filtered := make([]string, 0, len(tf.contents))
	for _, line := range tf.contents {
		if strings.HasPrefix(line, "+++") {
			filtered = append(filtered, strings.TrimPrefix(line, "+++"))
		} else if strings.HasPrefix(line, "---") {
			if !filter {
				filtered = append(filtered, strings.TrimPrefix(line, "---"))
			}
		} else {
			t.Fatalf("Test file line %q missing '+++' or '---' prefix", line)
		}
	}
	return strings.Join(filtered, "\n")
}

func TestFilterAndIndexCodeFile(t *testing.T) {
	for filePath, fileDesc := range testFiles {
		icf := filterAndIndexCodeFile(filePath, []byte(fileDesc.getContents(t, false)), false)
		if got, want := icf.CodeFile.Name, filePath; got != want {
			t.Errorf("Expected indexed file name %s, got %s", want, got)
		}
		if got, want := icf.CodeFile.Body, fileDesc.getContents(t, true); got != want {
			t.Errorf("Expected indexed file %s contents %q, got %q", filePath, want, got)
		}
		if got, want := icf.index, fileDesc.index; got != want {
			t.Errorf("Expected indexed file %s index %d, got %d", filePath, want, got)
		}
	}
}

// Writes testFiles to a temporary directory hierachy and passes the root path
// to the test function.
func testWithFiles(t *testing.T, test func(t *testing.T, dir string)) {
	dir, err := ioutil.TempDir("", "pg-test-bundler-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("Failed to remove temporary directory: %v", err)
		}
	}()
	for filePath, fileDesc := range testFiles {
		fileContents := fileDesc.getContents(t, false)
		fullPath := filepath.Join(dir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to mkdir %q: %v", filepath.Dir(filePath), err)
		}
		if err := ioutil.WriteFile(fullPath, []byte(fileContents), 0644); err != nil {
			t.Fatalf("Failed to write %q: %v", filePath, err)
		}
	}
	test(t, dir)
}

func TestBundleWalkFail(t *testing.T) {
	testWithFiles(t, func(t *testing.T, dir string) {
		// rootPath points to a nonexistent dir
		if _, err := MakeBundle(filepath.Join(dir, "nosuchdir"), []string{"*"}, true); err == nil {
			t.Errorf("Expected bundling to fail with rootPath pointing to a nonexistent dir")
		}

		// rootPath points to a regular file, not dir
		regularFilePath := filepath.Join(dir, "regular.txt")
		if err := ioutil.WriteFile(regularFilePath, []byte("foobar"), 0755); err != nil {
			t.Fatalf("Failed writing dummy regular file: %v", err)
		}
		if _, err := MakeBundle(filepath.Join(dir, regularFilePath), []string{"*"}, true); err == nil {
			t.Errorf("Expected bundling to fail with rootPath pointing to a regular file")
		}

		// root dir contains inaccessible dir
		protectedDirPath := filepath.Join(dir, "alpha", "secrets")
		if err := os.MkdirAll(protectedDirPath, 0000); err != nil {
			t.Fatalf("Failed creating dummy read-protected dir: %v", err)
		}
		if _, err := MakeBundle(dir, []string{"*"}, true); err == nil {
			t.Errorf("Expected bundling to fail with read-protected dir in rootPath tree")
		}
	})
}

func printBundleFiles(t *testing.T, bundle *bundle.Bundle) {
	gotFiles := make([]string, 0, len(bundle.Files))
	for _, file := range bundle.Files {
		gotFiles = append(gotFiles, file.Name)
	}
	t.Logf("Bundle files: %s.", strings.Join(gotFiles, "; "))
}

func printExpectFiles(t *testing.T, expectFiles []string) {
	t.Logf("Expect files: %s.", strings.Join(expectFiles, "; "))
}

// Checks that the bundle contains files in expectFiles, in the same order,
// with correctly filtered contents.
func checkBundleFiles(t *testing.T, bundle *bundle.Bundle, prefix string, expectFiles []string, empty bool) {
	if got, want := len(bundle.Files), len(expectFiles); got != want {
		t.Errorf("Expected %d files in bundle, got %d", want, got)
		printBundleFiles(t, bundle)
		printExpectFiles(t, expectFiles)
		return
	}
	for i, file := range bundle.Files {
		if got, want := file.Name, expectFiles[i]; got != want {
			t.Errorf("Bundle file mismatch at position %d: expected %s, got %s", i, want, got)
			printBundleFiles(t, bundle)
			printExpectFiles(t, expectFiles)
			return
		}
		if empty {
			if len(file.Body) > 0 {
				t.Errorf("Expected bundle file %s to be empty", file.Name)
			}
		} else {
			expectFileDesc, ok := testFiles[filepath.Join(prefix, expectFiles[i])]
			if !ok {
				t.Fatalf("Unknown expected bundle file %s for prefix %s", expectFiles[i], prefix)
			} else if got, want := file.Body, expectFileDesc.getContents(t, true); got != want {
				t.Errorf("Expected bundle file %s contents %q, got %q", want, got)
			}
		}
	}
}

// Makes a bundle rooted in dir+prefix, filtered by globList. Expects bundle
// to contain files in expectFiles, in the same order. If expectFiles is nil,
// expects bundling to fail.
func runBundle(t *testing.T, dir, prefix string, globList []string, expectFiles *[]string, empty bool) {
	bundle, err := MakeBundle(filepath.Join(dir, prefix), globList, empty)
	if expectFiles == nil {
		if err == nil {
			t.Errorf("Expected bundling to fail for prefix %q, globList %v", prefix, globList)
			printBundleFiles(t, bundle)
		}
	} else {
		if err != nil {
			t.Errorf("Expected bundling to succeed for prefix %q, globList %v, got error: %v", prefix, globList, err)
		} else {
			checkBundleFiles(t, bundle, prefix, *expectFiles, empty)
		}
	}
}

func TestMakeBundle(t *testing.T) {
	testWithFiles(t, func(t *testing.T, dir string) {
		// all files, test proper sorting
		runBundle(t, dir, "", []string{
			"*",
		}, &[]string{
			"alpha/one/foo/C-1.a",
			"beta/E-2.b",
			"alpha/one/B-3.b",
			"beta/two/G-4",
			"alpha/one/A-5.a",
			"alpha/two/D-6.d",
			"beta/one/F-7.a",
		}, true)

		// no files, empty pattern
		runBundle(t, dir, "", []string{}, &[]string{}, false)

		// all files containing 'one' in path, suffix match is implied
		runBundle(t, dir, "", []string{
			"one/**",
		}, &[]string{
			"alpha/one/foo/C-1.a",
			"alpha/one/B-3.b",
			"alpha/one/A-5.a",
			"beta/one/F-7.a",
		}, false)

		// more complex patterns
		runBundle(t, dir, "", []string{
			"alpha/*/*.*",
			"beta/*",
			"*-?",
		}, &[]string{
			"beta/E-2.b",
			"alpha/one/B-3.b",
			"beta/two/G-4",
			"alpha/one/A-5.a",
			"alpha/two/D-6.d",
		}, true)

		// files matched by multiple patterns should be included once
		runBundle(t, dir, "", []string{
			"*.a",
			"beta/**",
		}, &[]string{
			"alpha/one/foo/C-1.a",
			"beta/E-2.b",
			"beta/two/G-4",
			"alpha/one/A-5.a",
			"beta/one/F-7.a",
		}, false)

		// pattern matches no regular files (but matches a folder), bundling fails
		runBundle(t, dir, "", []string{
			"foo",
		}, nil, false)

		// only one of the patterns matches no files, bundling still fails
		runBundle(t, dir, "", []string{
			"alpha/*/*.*",
			"beta/*",
			"idontexist",
			"*-?",
		}, nil, false)

		// pathnames relative to bundling root
		runBundle(t, dir, "alpha", []string{
			"two/**",
			"*.a",
		}, &[]string{
			"one/foo/C-1.a",
			"one/A-5.a",
			"two/D-6.d",
		}, false)

		// globs match relative to bundling root, so this fails
		runBundle(t, dir, "alpha", []string{
			"alpha/two/**",
		}, nil, false)
	})
}
