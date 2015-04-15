// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// JsonSink is an Event Sink that serializes written Events to JSON and writes
// them one per line.
//
// JsonSink.Write is thread-safe. The underlying io.Writer is flushed after
// every write, if it supports flushing. Optionally filters out debug Events.

package event

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// Initialize using NewJsonSink.
type JsonSink struct {
	filterDebug bool

	mu sync.Mutex
	w  io.Writer
}

var _ Sink = (*JsonSink)(nil)

func NewJsonSink(writer io.Writer, filterDebug bool) *JsonSink {
	return &JsonSink{
		w:           writer,
		filterDebug: filterDebug,
	}
}

func (es *JsonSink) Write(events ...Event) error {
	if es.filterDebug {
		events = filter(events...)
	}
	evJson, err := jsonize(events...)
	if err != nil {
		return err
	}
	return es.writeJson(evJson...)
}

// Filters out debug Events.
func filter(events ...Event) []Event {
	filtered := make([]Event, 0, len(events))
	for _, ev := range events {
		if ev.Stream != "debug" {
			filtered = append(filtered, ev)
		}
	}
	return filtered
}

// Converts Events to JSON.
func jsonize(events ...Event) (evJson [][]byte, err error) {
	evJson = make([][]byte, 0, len(events))
	for _, ev := range events {
		var js []byte
		js, err = json.Marshal(&ev)
		if err != nil {
			return
		}
		evJson = append(evJson, js)
	}
	return
}

// Writes JSON lines and flushes output.
func (es *JsonSink) writeJson(evJson ...[]byte) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	defer es.flush()
	for _, js := range evJson {
		_, err := es.w.Write(append(js, '\n'))
		if err != nil {
			return err
		}
	}
	return nil
}

func (es *JsonSink) flush() {
	if f, ok := es.w.(http.Flusher); ok {
		f.Flush()
	}
}
