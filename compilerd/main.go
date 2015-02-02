// HTTP server for saving, loading, and executing playground examples.

package main

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var (
	// This channel is closed when the server begins shutting down.
	// No values are ever sent to it.
	lameduck chan bool = make(chan bool)

	address = flag.String("address", ":8181", "Address to listen on.")

	// Note, shutdown triggers on SIGTERM or when the time limit is hit.
	enableShutdown = flag.Bool("shutdown", true, "Whether to ever shutdown the machine.")

	// Maximum request and output size. Same limit as imposed by Go tour.
	// Note: The response includes error and status messages as well as output,
	// so it can be larger (usually by a small constant, hard limited to
	// 2*maxSize).
	// maxSize should be large enough to fit all error and status messages
	// written by compilerd to prevent reaching the hard limit.
	maxSize = 1 << 16
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

//////////////////////////////////////////
// HTTP server

func healthz(w http.ResponseWriter, r *http.Request) {
	select {
	case <-lameduck:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.Write([]byte("ok"))
	}
}

func main() {
	flag.Parse()

	if err := seedRNG(); err != nil {
		panic(err)
	}

	if *enableShutdown {
		limit_min := 60
		delay_min := limit_min/2 + rand.Intn(limit_min/2)

		// VMs will be periodically killed to prevent any owned VMs from causing
		// damage. We want to shutdown cleanly before then so we don't cause
		// requests to fail.
		go waitForShutdown(time.Minute * time.Duration(delay_min))
	}

	if err := initDBHandles(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/compile", handlerCompile)
	http.HandleFunc("/load", handlerLoad)
	http.HandleFunc("/save", handlerSave)

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

// Checks if the POST method was used and returns the request body.
// Returns nil iff response processing should not continue.
func getPostBody(w http.ResponseWriter, r *http.Request) []byte {
	if r.Body == nil || r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	return buf.Bytes()
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
