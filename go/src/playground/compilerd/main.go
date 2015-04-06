// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// HTTP server for saving, loading, and executing playground examples.

// TODO(nlacasse,ivanpi): The word "compile" is no longer appropriate for what
// this server does. Rename to something better.

package main

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}

var (
	// This channel is closed when clean exit is triggered.
	// No values are ever sent to it.
	lameduck chan bool = make(chan bool)

	address = flag.String("address", ":8181", "Address to listen on.")

	// compilerd exits cleanly on SIGTERM or after a random amount of time,
	// between listenTimeout/2 and listenTimeout.
	listenTimeout = flag.Duration("listen-timeout", 60*time.Minute, "Maximum amount of time to listen for before exiting. A value of 0 disables the timeout.")

	// Maximum request and output size.
	// 1<<16 is the same limit as imposed by Go tour.
	// Note: The response includes error and status messages as well as output,
	// so it can be larger (usually by a small constant, hard limited to
	// 2*maxSize).
	// maxSize should be large enough to fit all error and status messages
	// written by compilerd to prevent reaching the hard limit.
	maxSize = flag.Int("max-size", 1<<16, "Maximum request and output size.")

	// Maximum time to finish serving currently running requests before exiting
	// cleanly. No new requests are accepted during this time.
	exitDelay = 30 * time.Second
)

// Seeds the non-secure random number generator.
func seedRNG() error {
	var seed int64
	err := binary.Read(crand.Reader, binary.LittleEndian, &seed)
	if err != nil {
		return fmt.Errorf("reseed failed: %v", err)
	}
	rand.Seed(seed)
	return nil
}

func main() {
	flag.Parse()

	if err := seedRNG(); err != nil {
		panic(err)
	}

	c := newCompiler()

	listenForNs := listenTimeout.Nanoseconds()
	if listenForNs > 0 {
		delayNs := listenForNs/2 + rand.Int63n(listenForNs/2)

		// VMs will be periodically killed to prevent any owned VMs from causing
		// damage. We want to exit cleanly before then so we don't cause requests
		// to fail. When compilerd exits, a watchdog will shut the machine down
		// after a short delay.
		go waitForExit(c, time.Nanosecond*time.Duration(delayNs))
	}

	if err := initDBHandles(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/compile", c.handlerCompile)
	http.HandleFunc("/load", handlerLoad)
	http.HandleFunc("/save", handlerSave)
	http.HandleFunc("/healthz", handlerHealthz)

	log.Printf("Serving %s\n", *address)
	if err := http.ListenAndServe(*address, nil); err != nil {
		panic(err)
	}
}

func waitForExit(c *compiler, limit time.Duration) {
	// Exit if we get a SIGTERM.
	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM)

	// Or if the time limit expires.
	deadline := time.After(limit)
	log.Println("Exiting at", time.Now().Add(limit))
Loop:
	for {
		select {
		case <-deadline:
			log.Println("Deadline expired, exiting in at most", exitDelay)
			break Loop
		case <-term:
			log.Println("Got SIGTERM, exiting in at most", exitDelay)
			break Loop
		}
	}

	// Fail health checks so we stop getting requests.
	close(lameduck)

	go func() {
		select {
		case <-time.After(exitDelay):
			fmt.Errorf("Dispatcher did not stop in %v, exiting.", exitDelay)
			os.Exit(1)
		}
	}()

	// Stop the compiler and wait for all in-progress jobs to finish.
	c.stop()

	// Give the server some extra time to send any remaning responses that are
	// queued to be sent.
	time.Sleep(2 * time.Second)

	os.Exit(0)
}

//////////////////////////////////////////
// HTTP request helpers

// Handles CORS options and pre-flight requests.
// Returns false iff response processing should not continue.
func handleCORS(w http.ResponseWriter, r *http.Request) bool {
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
		return false
	}

	return true
}

// Checks if the GET method was used.
// Returns false iff response processing should not continue.
func checkGetMethod(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	return true
}

// Checks if the POST method was used and returns the first limit bytes of the
// request body.
// Returns nil iff response processing should not continue.
func getPostBody(w http.ResponseWriter, r *http.Request, limit int) []byte {
	if r.Body == nil || r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(io.LimitReader(r.Body, int64(limit)))
	return buf.Bytes()
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	select {
	case <-lameduck:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.Write([]byte("ok"))
	}
}

//////////////////////////////////////////
// Shared helper functions

func stringHash(data []byte) string {
	hv := rawHash(data)
	return hex.EncodeToString(hv[:])
}

func rawHash(data []byte) [32]byte {
	return sha256.Sum256(data)
}
