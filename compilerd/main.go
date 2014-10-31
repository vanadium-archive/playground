package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/groupcache/lru"

	"veyron.io/playground/event"
)

type ResponseBody struct {
	Errors string
	Events []event.Event
}

type CachedResponse struct {
	Status int
	Body   ResponseBody
}

var (
	// This channel is closed when the server begins shutting down.
	// No values are ever sent to it.
	lameduck chan bool = make(chan bool)

	address = flag.String("address", ":8181", "Address to listen on.")

	// Note, shutdown triggers on SIGTERM or when the time limit is hit.
	enableShutdown = flag.Bool("shutdown", true, "Whether to ever shutdown the machine.")

	useDocker = flag.Bool("use-docker", true, "Whether to use Docker to run builder; if false, we run the builder directly.")

	// Maximum request and response size. Same limit as imposed by Go tour.
	maxSize = 1 << 16

	// In-memory LRU cache of request/response bodies. Keys are sha1 sum of
	// request bodies (20 bytes each), values are of type CachedResponse.
	// NOTE(nlacasse): The cache size (10k) was chosen arbitrarily and should
	// perhaps be optimized.
	cache = lru.New(10000)
)

func healthz(w http.ResponseWriter, r *http.Request) {
	select {
	case <-lameduck:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.Write([]byte("ok"))
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	// CORS headers.
	// TODO(nlacasse): Fill the origin header in with actual playground origin
	// before going to production.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding")

	// CORS sends an OPTIONS pre-flight request to make sure the request will be
	// allowed.
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Body == nil || r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	wantDebug := r.FormValue("debug") == "1"
	requestBody := streamToBytes(r.Body)

	if len(requestBody) > maxSize {
		responseBody := new(ResponseBody)
		responseBody.Errors = "Program too large."
		respondWithBody(w, http.StatusBadRequest, responseBody, wantDebug)
		return
	}

	// Hash the body and see if it's been cached. If so, return the cached
	// response status and body.
	// NOTE(sadovsky): In the client we may shift timestamps (based on current
	// time) and introduce a fake delay.
	requestBodyHash := sha1.Sum(requestBody)
	if cachedResponse, ok := cache.Get(requestBodyHash); ok {
		if cachedResponseStruct, ok := cachedResponse.(CachedResponse); ok {
			respondWithBody(w, cachedResponseStruct.Status, &cachedResponseStruct.Body, wantDebug)
			return
		} else {
			log.Printf("Invalid cached response: %v\n", cachedResponse)
			cache.Remove(requestBodyHash)
		}
	}

	// TODO(nlacasse): It would be cool if we could stream the output
	// messages while the process is running, rather than waiting for it to
	// exit and dumping all the output then.

	id := <-uniq

	// TODO(sadovsky): Set runtime constraints on CPU and memory usage.
	// http://docs.docker.com/reference/run/#runtime-constraints-on-cpu-and-memory
	var cmd *exec.Cmd
	if *useDocker {
		cmd = docker("run", "-i", "--name", id, "playground")
	} else {
		cmd = exec.Command("builder")
	}
	cmd.Stdin = bytes.NewReader(requestBody)

	// Builder will return all normal output as json events on stdout, and will
	// return unexpected errors on stderr.
	// TODO(sadovsky): Security issue: what happens if the program output is huge?
	// We can restrict memory use of the Docker container, but these buffers are
	// outside Docker.
	stdoutBuf, stderrBuf := new(bytes.Buffer), new(bytes.Buffer)
	cmd.Stdout, cmd.Stderr = stdoutBuf, stderrBuf

	// Arbitrary deadline (enough to compile, run, shutdown).
	// TODO(sadovsky): For now this is set high to avoid spurious timeouts.
	// Playground execution speed needs to be optimized.
	timeout := time.After(10 * time.Second)
	timedOut := false

	exit := make(chan error)
	go func() { exit <- cmd.Run() }()

	select {
	case <-exit:
	case <-timeout:
		// NOTE(sadovsky): More builder output could show up on stdout after this
		// message, but that's not really a problem.
		stderrBuf.Write([]byte("\nTime exceeded; killing...\n"))
		timedOut = true
	}

	// If the response is bigger than the limit, create an "output too large"
	// error response, cache it, and return it.
	if stdoutBuf.Len() > maxSize {
		status := http.StatusBadRequest
		responseBody := new(ResponseBody)
		responseBody.Errors = "Program output too large."
		cache.Add(requestBodyHash, CachedResponse{
			Status: status,
			Body:   *responseBody,
		})
		respondWithBody(w, status, responseBody, wantDebug)
		return
	}

	responseBody := new(ResponseBody)
	// TODO(nlacasse): Make these errors Events, so that we can send them
	// back in the Events array.  This will simplify streaming the events to the
	// client in realtime.
	responseBody.Errors = stderrBuf.String()

	// Decode the json events from stdout and add them to the responseBody.
	for line, err := stdoutBuf.ReadBytes('\n'); err == nil; line, err = stdoutBuf.ReadBytes('\n') {
		var e event.Event
		json.Unmarshal(line, &e)
		responseBody.Events = append(responseBody.Events, e)
	}

	// If we timed out, do not cache anything.
	// TODO(sadovsky): This policy is helpful for development, but may not be wise
	// for production. Revisit.
	if !timedOut {
		cache.Add(requestBodyHash, CachedResponse{
			Status: http.StatusOK,
			Body:   *responseBody,
		})
	}
	respondWithBody(w, http.StatusOK, responseBody, wantDebug)

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

func respondWithBody(w http.ResponseWriter, status int, body *ResponseBody, wantDebug bool) {
	// If the request does not include query param debug=true, strip any debug
	// events produced by the builder. Note, these events don't contain any
	// sensitive information, so guarding with a query parameter is sufficient.
	if !wantDebug {
		// TODO(sadovsky): Use pointers to avoid copying Event structs, or cache
		// debug-stripped responses in addition to full responses.
		eventsNoDebug := make([]event.Event, 0, len(body.Events))
		for _, e := range body.Events {
			if e.Stream != "debug" {
				eventsNoDebug = append(eventsNoDebug, e)
			}
		}
		body.Events = eventsNoDebug
	}

	bodyJson, _ := json.Marshal(body)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", len(bodyJson)))
	w.Write(bodyJson)

	// TODO(nlacasse): This flush doesn't really help us right now, but we'll
	// definitely need something like it when we switch to the streaming model.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	} else {
		log.Println("Cannot flush.")
	}
}

func streamToBytes(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
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

func main() {
	flag.Parse()

	if *enableShutdown {
		limit_min := 60
		delay_min := limit_min/2 + rand.Intn(limit_min/2)

		// VMs will be periodically killed to prevent any owned VMs from causing
		// damage. We want to shutdown cleanly before then so we don't cause
		// requests to fail.
		go waitForShutdown(time.Minute * time.Duration(delay_min))
	}

	http.HandleFunc("/compile", handler)
	http.HandleFunc("/healthz", healthz)

	log.Printf("Serving %s\n", *address)
	http.ListenAndServe(*address, nil)
}

func waitForShutdown(limit time.Duration) {
	var beforeExit func() error

	// Shutdown if we get a SIGTERM.
	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM)

	// Or if the time limit expires.
	deadline := time.After(limit)
	log.Println("Shutting down at", time.Now().Add(limit))
Loop:
	for {
		select {
		case <-deadline:
			// Shutdown the VM.
			log.Println("Deadline expired, shutting down.")
			beforeExit = exec.Command("sudo", "halt").Run
			break Loop
		case <-term:
			log.Println("Got SIGTERM, shutting down.")
			// VM is probably already shutting down, so just exit.
			break Loop
		}
	}

	// Fail health checks so we stop getting requests.
	close(lameduck)

	// Give running requests time to finish.
	time.Sleep(30 * time.Second)

	// Go ahead and shutdown.
	if beforeExit != nil {
		err := beforeExit()
		if err != nil {
			panic(err)
		}
	}
	os.Exit(0)
}
