// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Representation of a playground bundle.
// A bundle consists of a set of code files in a hierarchical file system. File
// type is inferred from the extension.

package bundle

// TODO(ivanpi): Add validity check (file extensions, etc) and refactor builder
// and storage to use the same structure.

type Bundle struct {
	Files []*CodeFile `json:"files"`
	// TODO(ivanpi): Add slug, title, description? Merge with compilerd.BundleFullResponse?
}

type CodeFile struct {
	Name string `json:"name"`
	Body string `json:"body"`
}
