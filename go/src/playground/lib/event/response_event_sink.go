// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"playground/lib"
)

// Initialize using NewResponseEventSink.
// An event.Sink which also saves all written Events regardless of successful
// writes to the underlying ResponseWriter.
type ResponseEventSink struct {
	// The mutex is used to ensure the same sequence of events being written to
	// both the JsonSink and the written Event array.
	mu sync.Mutex
	JsonSink
	written []Event
}

func NewResponseEventSink(writer io.Writer, filterDebug bool) *ResponseEventSink {
	return &ResponseEventSink{
		JsonSink: *NewJsonSink(writer, filterDebug),
	}
}

func (r *ResponseEventSink) Write(events ...Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.written = append(r.written, events...)
	return r.JsonSink.Write(events...)
}

// Returns and clears the history of Events written to the ResponseEventSink.
func (r *ResponseEventSink) PopWrittenEvents() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.written
	r.written = nil
	return events
}

// Each line written to the returned writer, up to limit bytes total, is parsed
// into an Event and written to Sink.
// If the limit is reached or an invalid line read, the corresponding callback
// is called and the relay stopped.
// The returned stop() function stops the relaying.
func LimitedEventRelay(sink Sink, limit int, limitCallback func(), errorCallback func(err error)) (writer io.Writer, stop func()) {
	pipeReader, pipeWriter := io.Pipe()
	done := make(chan bool)
	stop = lib.DoOnce(func() {
		// Closing the pipe will cause the main relay loop to stop reading (EOF).
		// Writes will fail with ErrClosedPipe.
		pipeReader.Close()
		pipeWriter.Close()
		// Wait for the relay goroutine to finish.
		<-done
	})
	writer = lib.NewLimitedWriter(pipeWriter, limit, func() {
		limitCallback()
		stop()
	})
	go func() {
		bufr := bufio.NewReaderSize(pipeReader, limit)
		var line []byte
		var err error
		// Relay complete lines (events) until EOF or a read error is encountered.
		for line, err = bufr.ReadBytes('\n'); err == nil; line, err = bufr.ReadBytes('\n') {
			var e Event
			err = json.Unmarshal(line, &e)
			if err != nil {
				err = fmt.Errorf("failed unmarshalling event: %q", line)
				break
			}
			sink.Write(e)
		}
		if err != io.EOF && err != io.ErrClosedPipe {
			errorCallback(err)
			// Use goroutine to prevent deadlock on done channel.
			go stop()
		}
		done <- true
	}()
	return
}
