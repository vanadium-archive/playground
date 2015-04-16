// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// MultiWriter creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
//
// Similar to http://golang.org/src/pkg/io/multi.go.

package lib

import (
	"io"
	"sync"

	"v.io/x/playground/lib/log"
)

// Initialize using NewMultiWriter.
type MultiWriter struct {
	writers []io.Writer
	mu      sync.Mutex
	wrote   bool
}

var _ io.Writer = (*MultiWriter)(nil)

func NewMultiWriter() *MultiWriter {
	return &MultiWriter{writers: []io.Writer{}}
}

// Returns self for convenience.
func (t *MultiWriter) Add(w io.Writer) *MultiWriter {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.wrote {
		log.Panic("Tried to add writer after data has been written.")
	}
	t.writers = append(t.writers, w)
	return t
}

func (t *MultiWriter) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	t.wrote = true
	t.mu.Unlock()
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}
