// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Handler for HTTP requests to compile and run playground examples.
//
// handlerCompile() handles a POST request with bundled example source code.
// The bundle is passed to the builder command, which is run inside a Docker
// sandbox. Builder output is streamed back to the client in realtime and
// cached.

package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/golang/groupcache/lru"

	"playground/compilerd/jobqueue"
	"playground/lib"
	"playground/lib/event"
	"playground/lib/hash"
)

var (
	// In-memory LRU cache of request/response bodies. Keys are sha256 sums of
	// request bodies (32 bytes each), values are of type cachedResponse.
	// NOTE(nlacasse): The cache size (10k) was chosen arbitrarily and should
	// perhaps be optimized.
	cache = lru.New(10000)

	useDocker = flag.Bool("use-docker", true, "Whether to use Docker to run builder; if false, we run the builder directly.")

	// TODO(nlacasse): Experiment with different values for parallelism and
	// dockerMemLimit once we have performance testing.
	// There should be some memory left over for the system, http server, and
	// docker daemon. The GCE n1-standard machines have 3.75GB of RAM, so the
	// default value below should leave plenty of room.
	parallelism    = flag.Int("parallelism", 5, "Maximum number of builds to run in parallel.")
	dockerMemLimit = flag.Int("total-docker-memory", 3000, "Total memory limit for all Docker build instances in MB.")

	// Arbitrary deadline (enough to compile, run, shutdown).
	// TODO(sadovsky): For now this is set high to avoid spurious timeouts.
	// Playground execution speed needs to be optimized.
	maxTime = flag.Duration("max-time", 10*time.Second, "Maximum time for build to run.")

	// TODO(nlacasse): The default value of 100 was chosen arbitrarily and
	// should be tuned.
	jobQueueCap = flag.Int("job-queue-capacity", 100, "Maximum number of jobs to allow in the job queue. Attempting to add a new job will fail if the queue is full.")
)

// cachedResponse is the type of values stored in the lru cache.
type cachedResponse struct {
	Status int
	Events []event.Event
}

// compiler handles compile requests by enqueuing them on the dispatcher's
// work queue.
type compiler struct {
	dispatcher jobqueue.Dispatcher
}

// newCompiler creates a new compiler.
func newCompiler() *compiler {
	return &compiler{
		dispatcher: jobqueue.NewDispatcher(*parallelism, *jobQueueCap),
	}
}

// POST request that returns cached results if any exist, otherwise schedules
// the bundle to be run and caches the results.
func (c *compiler) handlerCompile(w http.ResponseWriter, r *http.Request) {
	if !handleCORS(w, r) {
		return
	}

	// Check method and read POST body.
	// Limit is set to maxSize+1 to allow distinguishing between exactly maxSize
	// and larger than maxSize requests.
	requestBody := getPostBody(w, r, *maxSize+1)
	if requestBody == nil {
		return
	}

	// If the request does not include query param debug=true, strip any debug
	// events produced by the builder. Note, these events don't contain any
	// sensitive information, so guarding with a query parameter is sufficient.
	wantDebug := r.FormValue("debug") == "1"

	openResponse := func(status int) *event.ResponseEventSink {
		w.Header().Add("Content-Type", "application/json")
		// No Content-Length, using chunked encoding.
		w.WriteHeader(status)
		// The response is hard limited to 2*maxSize: maxSize for builder stdout,
		// and another maxSize for compilerd error and status messages.
		return event.NewResponseEventSink(lib.NewLimitedWriter(w, 2*(*maxSize), lib.DoOnce(func() {
			log.Println("Hard response size limit reached.")
		})), !wantDebug)
	}

	if len(requestBody) > *maxSize {
		res := openResponse(http.StatusBadRequest)
		res.Write(event.New("", "stderr", "Program too large."))
		return
	}

	// Hash the body and see if it's been cached. If so, return the cached
	// response status and body.
	// NOTE(sadovsky): In the client we may shift timestamps (based on current
	// time) and introduce a fake delay.
	requestBodyHash := hash.Raw(requestBody)
	if cr, ok := cache.Get(requestBodyHash); ok {
		if cachedResponseStruct, ok := cr.(cachedResponse); ok {
			res := openResponse(cachedResponseStruct.Status)
			event.Debug(res, "Sending cached response")
			res.Write(cachedResponseStruct.Events...)
			return
		} else {
			log.Panicf("Invalid cached response: %v\n", cr)
		}
	}

	res := openResponse(http.StatusOK)

	// Calculate memory limit for the docker instance running this job.
	dockerMem := *dockerMemLimit
	if *parallelism > 0 {
		dockerMem /= *parallelism
	}

	// Create a new compile job and queue it.
	job := jobqueue.NewJob(requestBody, res, *maxSize, *maxTime, *useDocker, dockerMem)
	resultChan, err := c.dispatcher.Enqueue(job)
	if err != nil {
		// TODO(nlacasse): This should send a StatusServiceUnavailable, not a StatusOK.
		res.Write(event.New("", "stderr", "Service busy. Please try again later."))
		return
	}

	// Go's httptest.NewRecorder does not support http.CloseNotifier, so we
	// can't assume that w.(httpCloseNotifier) will succeed.
	var clientDisconnect <-chan bool
	if closeNotifier, ok := w.(http.CloseNotifier); ok {
		clientDisconnect = closeNotifier.CloseNotify()
	}

	// Wait for job to finish and cache results if job succeeded.
	// We do this in a for loop because we always need to wait for the result
	// before closing the http handler, since Go panics when writing to the
	// response after the handler has exited.
	for {
		select {
		case <-clientDisconnect:
			// If the client disconnects before job finishes, cancel the job.
			// If job has already started, the job will finish and the results
			// will be cached.
			log.Printf("Client disconnected. Cancelling job.")
			job.Cancel()
		case result := <-resultChan:
			if result.Success {
				event.Debug(res, "Caching response")
				cache.Add(requestBodyHash, cachedResponse{
					Status: http.StatusOK,
					Events: result.Events,
				})
			} else {
				event.Debug(res, "Internal errors encountered, not caching response")
			}
			return
		}
	}
}

// stop waits for any in-progress jobs to finish, and cancels any jobs that
// have not started running yet.
func (c *compiler) stop() {
	c.dispatcher.Stop()
}
