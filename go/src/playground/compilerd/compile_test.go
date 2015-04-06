// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"playground/compilerd/jobqueue"
	"playground/lib/event"
)

// mockDispatcher implements the jobqueue.Dispatcher interface.
type mockDispatcher struct {
	jobs        []*jobqueue.Job
	sendSuccess bool
}

// Enqueue responds to every job after 100ms. The only event message will
// contain the job body. The result will be a success if "sendSuccess" is true.
func (d *mockDispatcher) Enqueue(j *jobqueue.Job) (chan jobqueue.Result, error) {
	d.jobs = append(d.jobs, j)

	e := event.Event{
		Message: string(j.Body()),
	}

	result := jobqueue.Result{
		Success: d.sendSuccess,
		Events:  []event.Event{e},
	}

	resultChan := make(chan jobqueue.Result)

	go func() {
		time.Sleep(100 * time.Millisecond)
		resultChan <- result
	}()

	return resultChan, nil
}

func (d *mockDispatcher) Stop() {
}

var _ = jobqueue.Dispatcher((*mockDispatcher)(nil))

func sendCompileRequest(c *compiler, method string, body io.Reader) *httptest.ResponseRecorder {
	path := "/compile"
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		panic(err)
	}

	w := httptest.NewRecorder()
	c.handlerCompile(w, req)
	return w
}

func TestNonPostMethodsAreBadRequests(t *testing.T) {
	c := &compiler{}
	body := bytes.NewBufferString("foobar")
	methods := []string{"GET", "PUT", "DELETE", "FOOBAR"}
	for _, method := range methods {
		w := sendCompileRequest(c, method, body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected HTTP method %v to result in status %v but got %v", method, http.StatusBadRequest, w.Code)
		}
	}
}

func TestEmptyBodyIsBadRequest(t *testing.T) {
	c := &compiler{}
	w := sendCompileRequest(c, "POST", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected POST with empty body to result in status %v but got %v", http.StatusBadRequest, w.Code)
	}
}

func TestLargeBodyIsBadRequest(t *testing.T) {
	c := &compiler{}
	body := bytes.NewBuffer(make([]byte, *maxSize+1))
	w := sendCompileRequest(c, "POST", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected POST with large body to result in status %v but got %v", http.StatusBadRequest, w.Code)
	}
}

func TestSuccessResultsAreCached(t *testing.T) {
	dispatcher := &mockDispatcher{
		sendSuccess: true,
	}
	c := &compiler{
		dispatcher: dispatcher,
	}

	bodyString := "foobar"
	body := bytes.NewBufferString(bodyString)
	bodyBytes := body.Bytes()
	requestBodyHash := rawHash(bodyBytes)

	// Check that body is not already in cache.
	if _, ok := cache.Get(requestBodyHash); ok {
		t.Errorf("Expected request body not to be in cache, but it was.")
	}

	w := sendCompileRequest(c, "POST", body)
	if w.Code != http.StatusOK {
		t.Errorf("Expected POST with body %v to result in status %v but got %v", bodyString, http.StatusOK, w.Code)
	}

	// Check that the dispatcher queued the job.
	if len(dispatcher.jobs) != 1 {
		t.Errorf("Expected len(dispatcher.jobs) to be 1 but got %v", len(dispatcher.jobs))
	}

	// Check that body is now in the cache.
	if cr, ok := cache.Get(requestBodyHash); !ok {
		t.Errorf("Expected request body to be in cache, but it was not.")
	} else {
		cachedResponseStruct := cr.(cachedResponse)
		if cachedResponseStruct.Status != http.StatusOK {
			t.Errorf("Expected cached result status to be %v but got %v", http.StatusOK, cachedResponseStruct.Status)
		}
		want := string(bodyBytes)
		if len(cachedResponseStruct.Events) != 1 || cachedResponseStruct.Events[0].Message != want {
			t.Errorf("Expected cached result body to contain single event with message %v but got %v", want, cachedResponseStruct.Events)

		}

		// Check that the dispatcher did not queue the second request, since it was in the cache.
		if len(dispatcher.jobs) != 1 {
			t.Errorf("Expected len(dispatcher.jobs) to be 1 but got %v", len(dispatcher.jobs))
		}
	}
}

func TestErrorResultsAreNotCached(t *testing.T) {
	dispatcher := &mockDispatcher{
		sendSuccess: false,
	}

	c := &compiler{
		dispatcher: dispatcher,
	}

	bodyString := "bazbar"
	body := bytes.NewBufferString(bodyString)
	bodyBytes := body.Bytes()
	requestBodyHash := rawHash(bodyBytes)

	// Check that body is not already in cache.
	if _, ok := cache.Get(requestBodyHash); ok {
		t.Errorf("Expected request body not to be in cache, but it was.")
	}

	w := sendCompileRequest(c, "POST", body)
	if w.Code != http.StatusOK {
		t.Errorf("Expected POST with body %v to result in status %v but got %v", bodyString, http.StatusOK, w.Code)
	}

	// Check that the dispatcher queued the request.
	if len(dispatcher.jobs) != 1 {
		t.Errorf("Expected len(dispatcher.jobs) to be 1 but got %v", len(dispatcher.jobs))
	}

	// Check that body is still not in the cache.
	if _, ok := cache.Get(requestBodyHash); ok {
		t.Errorf("Expected request body not to be in cache, but it was.")
	}
}
