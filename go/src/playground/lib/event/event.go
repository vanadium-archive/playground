// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"time"
)

// Typed representation of data sent to stdin/stdout from a command.  These
// will be JSON-encoded and sent to the client.
type Event struct {
	// File associated with the command.
	File string
	// The text sent to stdin/stderr.
	Message string
	// Stream that the message was sent to, either "stdout" or "stderr".
	Stream string
	// Unix time, the number of nanoseconds elapsed since January 1, 1970 UTC.
	Timestamp int64
}

func New(file string, stream string, message string) Event {
	return Event{
		File:      file,
		Message:   message,
		Stream:    stream,
		Timestamp: time.Now().UnixNano(),
	}
}

// Stream for writing Events to.
type Sink interface {
	Write(events ...Event) error
}

func Debug(es Sink, args ...interface{}) {
	es.Write(New("", "debug", fmt.Sprintln(args...)))
}
