// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Implementation of io.Writer that streams each write as an Event to the
// wrapped Sink.

package event

import (
	"io"
)

// Initialize using NewStreamWriter.
type streamWriter struct {
	es         Sink
	fileName   string
	streamName string
}

var _ io.Writer = (*streamWriter)(nil)

func NewStreamWriter(es Sink, fileName, streamName string) *streamWriter {
	return &streamWriter{es: es, fileName: fileName, streamName: streamName}
}

func (ew *streamWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := ew.es.Write(New(ew.fileName, ew.streamName, string(p))); err != nil {
		return 0, err
	}
	return len(p), nil
}
