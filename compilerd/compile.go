package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"

	"v.io/playground/lib"
	"v.io/playground/lib/event"
)

type CachedResponse struct {
	Status int
	Events []event.Event
}

var (
	useDocker = flag.Bool("use-docker", true, "Whether to use Docker to run builder; if false, we run the builder directly.")

	// Arbitrary deadline (enough to compile, run, shutdown).
	// TODO(sadovsky): For now this is set high to avoid spurious timeouts.
	// Playground execution speed needs to be optimized.
	maxTime = 10 * time.Second

	// In-memory LRU cache of request/response bodies. Keys are sha1 sums of
	// request bodies (20 bytes each), values are of type CachedResponse.
	// NOTE(nlacasse): The cache size (10k) was chosen arbitrarily and should
	// perhaps be optimized.
	cache = lru.New(10000)
)

// POST request that compiles and runs the bundle and streams output to client.
func handlerCompile(w http.ResponseWriter, r *http.Request) {
	if !handleCORS(w, r) {
		return
	}

	// Check method and read POST body.
	requestBody := getPostBody(w, r)
	if requestBody == nil {
		return
	}

	// If the request does not include query param debug=true, strip any debug
	// events produced by the builder. Note, these events don't contain any
	// sensitive information, so guarding with a query parameter is sufficient.
	wantDebug := r.FormValue("debug") == "1"

	openResponse := func(status int) *responseEventSink {
		w.Header().Add("Content-Type", "application/json")
		// No Content-Length, using chunked encoding.
		w.WriteHeader(status)
		// The response is hard limited to 2*maxSize: maxSize for builder stdout,
		// and another maxSize for compilerd error and status messages.
		return newResponseEventSink(lib.NewLimitedWriter(w, 2*maxSize, lib.DoOnce(func() {
			log.Println("Hard response size limit reached.")
		})), !wantDebug)
	}

	if len(requestBody) > maxSize {
		res := openResponse(http.StatusBadRequest)
		res.Write(event.New("", "stderr", "Program too large."))
		return
	}

	// Hash the body and see if it's been cached. If so, return the cached
	// response status and body.
	// NOTE(sadovsky): In the client we may shift timestamps (based on current
	// time) and introduce a fake delay.
	requestBodyHash := sha1.Sum(requestBody)
	if cachedResponse, ok := cache.Get(requestBodyHash); ok {
		if cachedResponseStruct, ok := cachedResponse.(CachedResponse); ok {
			res := openResponse(cachedResponseStruct.Status)
			event.Debug(res, "Sending cached response")
			res.Write(cachedResponseStruct.Events...)
			return
		} else {
			log.Panicf("Invalid cached response: %v\n", cachedResponse)
		}
	}

	res := openResponse(http.StatusOK)

	id := <-uniq

	event.Debug(res, "Preparing to run program")

	// TODO(sadovsky): Set runtime constraints on CPU and memory usage.
	// http://docs.docker.com/reference/run/#runtime-constraints-on-cpu-and-memory
	var cmd *exec.Cmd
	if *useDocker {
		cmd = docker("run", "-i", "--name", id, "playground")
	} else {
		cmd = exec.Command("builder")
	}
	cmdKill := lib.DoOnce(func() {
		event.Debug(res, "Killing program")
		cmd.Process.Kill()
		if *useDocker {
			// Sudo doesn't pass sigkill to child processes, so we need to find and
			// kill the docker process directly.
			// The docker client can get in a state where stopping/killing/rm-ing
			// the container will not kill the client. The opposite should work
			// correctly (killing the docker client stops the container).
			// If not, the docker rm call below will.
			exec.Command("sudo", "pkill", "-SIGKILL", "-f", id).Run()
		}
	})

	cmd.Stdin = bytes.NewReader(requestBody)

	// Builder will return all normal output as JSON Events on stdout, and will
	// return unexpected errors on stderr.
	// TODO(sadovsky): Security issue: what happens if the program output is huge?
	// We can restrict memory use of the Docker container, but these buffers are
	// outside Docker.
	// TODO(ivanpi): Revisit above comment.
	sizedOut := false
	erroredOut := false

	userLimitCallback := func() {
		sizedOut = true
		cmdKill()
	}
	systemLimitCallback := func() {
		erroredOut = true
		cmdKill()
	}
	userErrorCallback := func(err error) {
		// A relay error can result from unparseable JSON caused by a builder bug
		// or a malicious exploit inside Docker. Panicking could lead to a DoS.
		log.Println(id, "builder stdout relay error:", err)
		erroredOut = true
		cmdKill()
	}

	outRelay, outStop := limitedEventRelay(res, maxSize, userLimitCallback, userErrorCallback)
	// Builder stdout should already contain a JSON Event stream.
	cmd.Stdout = outRelay

	// Any stderr is unexpected, most likely a bug (panic) in builder, but could
	// also result from a malicious exploit inside Docker.
	// It is quietly logged as long as it doesn't exceed maxSize.
	errBuffer := new(bytes.Buffer)
	cmd.Stderr = lib.NewLimitedWriter(errBuffer, maxSize, systemLimitCallback)

	event.Debug(res, "Running program")

	timeout := time.After(maxTime)
	// User code execution is time limited in builder.
	// This flag signals only unexpected timeouts. maxTime should be sufficient
	// for end-to-end request processing by builder for worst-case user input.
	// TODO(ivanpi): builder doesn't currently time compilation, so builder
	// worst-case execution time is not clearly bounded.
	timedOut := false

	exit := make(chan error)
	go func() { exit <- cmd.Run() }()

	select {
	case err := <-exit:
		if err != nil && !sizedOut {
			erroredOut = true
		}
	case <-timeout:
		timedOut = true
		cmdKill()
		<-exit
	}

	// Close and wait for the output relay.
	outStop()

	event.Debug(res, "Program exited")

	// Return the appropriate error message to the client.
	if timedOut {
		res.Write(event.New("", "stderr", "Internal timeout, please retry."))
	} else if erroredOut {
		res.Write(event.New("", "stderr", "Internal error, please retry."))
	} else if sizedOut {
		res.Write(event.New("", "stderr", "Program output too large, killed."))
	}

	// Log builder internal errors, if any.
	// TODO(ivanpi): Prevent caching? Report to client if debug requested?
	if errBuffer.Len() > 0 {
		log.Println(id, "builder stderr:", errBuffer.String())
	}

	event.Debug(res, "Response finished")

	// If we timed out or errored out, do not cache anything.
	// TODO(sadovsky): This policy is helpful for development, but may not be wise
	// for production. Revisit.
	if !timedOut && !erroredOut {
		cache.Add(requestBodyHash, CachedResponse{
			Status: http.StatusOK,
			Events: res.popWrittenEvents(),
		})
		event.Debug(res, "Caching response")
	} else {
		event.Debug(res, "Internal errors encountered, not caching response")
	}

	// TODO(nlacasse): This "docker rm" can be slow (several seconds), and seems
	// to block other Docker commands, thereby slowing down other concurrent
	// requests. We should figure out how to make it not block other Docker
	// commands. Setting GOMAXPROCS may or may not help.
	// See: https://github.com/docker/docker/issues/6480
	if *useDocker {
		go func() {
			docker("rm", "-f", id).Run()
		}()
	}
}

// Each line written to the returned writer, up to limit bytes total, is parsed
// into an Event and written to Sink.
// If the limit is reached or an invalid line read, the corresponding callback
// is called and the relay stopped.
// The returned stop() function stops the relaying.
func limitedEventRelay(sink event.Sink, limit int, limitCallback func(), errorCallback func(err error)) (writer io.Writer, stop func()) {
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
			var e event.Event
			err = json.Unmarshal(line, &e)
			if err != nil {
				err = fmt.Errorf("failed unmarshalling event: %s", line)
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

// Initialize using newResponseEventSink.
// An event.Sink which also saves all written Events regardless of successful
// writes to the underlying ResponseWriter.
type responseEventSink struct {
	// The mutex is used to ensure the same sequence of events being written to
	// both the JsonSink and the written Event array.
	mu sync.Mutex
	event.JsonSink
	written []event.Event
}

func newResponseEventSink(writer io.Writer, filterDebug bool) *responseEventSink {
	return &responseEventSink{
		JsonSink: *event.NewJsonSink(writer, filterDebug),
	}
}

func (r *responseEventSink) Write(events ...event.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.written = append(r.written, events...)
	return r.JsonSink.Write(events...)
}

// Returns and clears the history of Events written to the responseEventSink.
func (r *responseEventSink) popWrittenEvents() []event.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.written
	r.written = nil
	return events
}

func docker(args ...string) *exec.Cmd {
	fullArgs := []string{"docker"}
	fullArgs = append(fullArgs, args...)
	return exec.Command("sudo", fullArgs...)
}

// A channel which returns unique ids for the containers.
var uniq = make(chan string)

func init() {
	val := time.Now().UnixNano()
	go func() {
		for {
			uniq <- fmt.Sprintf("playground_%d", val)
			val++
		}
	}()
}
