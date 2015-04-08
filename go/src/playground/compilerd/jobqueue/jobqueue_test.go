// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jobqueue

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"playground/lib/event"
)

var (
	defaultMaxTime  = 10 * time.Second
	defaultMaxSize  = 1 << 16
	defaultMemLimit = 100
)

func init() {
	// Compile builder binary and put in path.
	pgDir := os.ExpandEnv("${V23_ROOT}/release/projects/playground/go")

	cmd := exec.Command("make", "builder")
	cmd.Dir = path.Join(pgDir, "src", "playground")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("Error running 'make builder'")
		fmt.Println(string(out))
		panic(err)
	}

	pgBinDir := path.Join(pgDir, "bin")
	if err := os.Setenv("PATH", pgBinDir+":"+os.Getenv("PATH")); err != nil {
		panic(err)
	}
}

// mockTestFile is a simple Go progam that will be sent to the dispatcher in the
// test runs.
var mockTestFile = `
package main

import(
	"fmt"
	"time"
)

func main() {
	fmt.Printf("PROGRAM START")
	time.Sleep(1 * time.Second)
	fmt.Printf("PROGRAM END")
}
`

// mockTestBody is a bundle containing the mockTestFile, as it would look if it
// came from an http request from the playground client.
var mockTestBody = []byte(fmt.Sprintf(`{
	"files": [ {
		"name": "src/main/main.go",
		"body": %v
	} ]
}`, strconv.Quote(mockTestFile)))

// testConfig encapsulates all the parameters and expectations for a single
// test run.
type testConfig struct {
	// Number of jobs to enqueue.
	jobs int

	// Dispatcher configuration.
	workers int
	jobCap  int

	// Job configuration.
	useDocker bool
	maxSize   int
	maxTime   time.Duration
	memLimit  int

	// Test expectations. Default is to expect success.
	expectEnqueueFail    bool
	expectOutputTooLarge bool
	expectJobFail        bool
}

func newMockResponseEventSink() *event.ResponseEventSink {
	var b bytes.Buffer
	return event.NewResponseEventSink(&b, false)
}

// eventsMatch returns true iff there is an event whose message matches the
// given string.
func eventsMatch(events []event.Event, match string) bool {
	for _, e := range events {
		if strings.Contains(e.Message, match) {
			return true
		}
	}
	return false
}

// assertExpectedResult asserts that the result matches the test expectations
// in the test config.
func assertExpectedResult(t *testing.T, c testConfig, r Result) {
	expectSuccess := !c.expectJobFail
	if expectSuccess != r.Success {
		t.Errorf("Expected result.Success to be %v but was %v. Test config: %#v", expectSuccess, r.Success, c)
	}

	if !r.Success {
		return
	}

	if c.expectOutputTooLarge {
		want := "Program output too large, killed."
		if !eventsMatch(r.Events, want) {
			t.Errorf("Event message %v not found in %#v. Test config: %#v", want, r.Events, c)
		}
		return
	}

	// Expect normal program output.
	want := "PROGRAM START"
	if !eventsMatch(r.Events, want) {
		t.Errorf("Event message %v not found in %#v. Test config: %#v", want, r.Events, c)
	}

	want = "PROGRAM END"
	if !eventsMatch(r.Events, want) {
		t.Errorf("Event message %v not found in %#v. Test config: %#v", want, r.Events, c)
	}
}

// runTest runs a particular test configuration. It starts a dispatcher, queues
// jobs and waits for them to finish, and asserts that they match the
// expectations in the testConfig.
func runTest(t *testing.T, c testConfig) {
	fmt.Printf("Testing %v jobs on %v workers with jobCap of %v and useDocker %v\n", c.jobs, c.workers, c.jobCap, c.useDocker)
	d := NewDispatcher(c.workers, c.jobCap)

	var enqueueError error

	// wg waits for all the jobs to finish.
	var wg sync.WaitGroup

	// Start all the jobs.
	for i := 0; i < c.jobs; i++ {
		res := newMockResponseEventSink()
		job := NewJob(mockTestBody, res, c.maxSize, c.maxTime, c.useDocker, c.memLimit)

		resultChan, err := d.Enqueue(job)
		if err != nil {
			enqueueError = err
			break
		}

		wg.Add(1)

		go func() {
			r := <-resultChan
			assertExpectedResult(t, c, r)
			wg.Done()
		}()
	}

	if !c.expectEnqueueFail && enqueueError != nil {
		t.Fatalf("Enqueue failed: %v. Test config: %#v", enqueueError, c)
	}

	if c.expectEnqueueFail && enqueueError == nil {
		t.Fatalf("Expected Enqueue to fail but it did not. Test config: %#v", c)
	}

	// Test must finish in 30 seconds.
	timeout := time.After(30 * time.Second)

	jobsFinished := make(chan bool)

	go func() {
		wg.Wait()
		jobsFinished <- true
	}()

	// Wait for timeout or all jobs to finish.
	select {
	case <-timeout:
		t.Fatalf("Expected jobs to complete but got timeout. Test config: %#v", c)
	case <-jobsFinished:
	}

	d.Stop()
}

func TestJobQueue(t *testing.T) {

	// Test success cases without docker.
	runTest(t, testConfig{
		jobs:      1,
		workers:   1,
		jobCap:    1,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		useDocker: false,
	})

	runTest(t, testConfig{
		jobs:      3,
		workers:   1,
		jobCap:    3,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		useDocker: false,
	})

	runTest(t, testConfig{
		jobs:      3,
		workers:   3,
		jobCap:    3,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		useDocker: false,
	})

	runTest(t, testConfig{
		jobs:      6,
		workers:   3,
		jobCap:    10,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		useDocker: false,
	})

	// Test success cases with docker.
	runTest(t, testConfig{
		jobs:      1,
		workers:   1,
		jobCap:    1,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		memLimit:  defaultMemLimit,
		useDocker: true,
	})

	runTest(t, testConfig{
		jobs:      3,
		workers:   1,
		jobCap:    3,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		memLimit:  defaultMemLimit,
		useDocker: true,
	})

	runTest(t, testConfig{
		jobs:      3,
		workers:   3,
		jobCap:    3,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		memLimit:  defaultMemLimit,
		useDocker: true,
	})

	runTest(t, testConfig{
		jobs:      6,
		workers:   3,
		jobCap:    10,
		maxSize:   defaultMaxSize,
		maxTime:   defaultMaxTime,
		memLimit:  defaultMemLimit,
		useDocker: true,
	})

	// Test Enqueue should fail when job capacity is exceeded.
	runTest(t, testConfig{
		jobs:              5,
		workers:           2,
		jobCap:            2,
		maxSize:           defaultMaxSize,
		maxTime:           defaultMaxTime,
		useDocker:         false,
		expectEnqueueFail: true,
	})

	runTest(t, testConfig{
		jobs:              5,
		workers:           2,
		jobCap:            2,
		maxSize:           defaultMaxSize,
		maxTime:           defaultMaxTime,
		memLimit:          defaultMemLimit,
		useDocker:         true,
		expectEnqueueFail: true,
	})

	// Test job should fail if it exceeds size limit.
	runTest(t, testConfig{
		jobs:                 1,
		workers:              1,
		jobCap:               1,
		maxSize:              100,
		maxTime:              defaultMaxTime,
		useDocker:            false,
		expectOutputTooLarge: true,
	})

	runTest(t, testConfig{
		jobs:                 1,
		workers:              1,
		jobCap:               1,
		maxSize:              100,
		maxTime:              defaultMaxTime,
		memLimit:             defaultMemLimit,
		useDocker:            true,
		expectOutputTooLarge: true,
	})

	// Test job should fail if it exceeds max time.
	runTest(t, testConfig{
		jobs:          1,
		workers:       1,
		jobCap:        1,
		maxSize:       defaultMaxSize,
		maxTime:       1 * time.Second,
		useDocker:     false,
		expectJobFail: true,
	})

	runTest(t, testConfig{
		jobs:          1,
		workers:       1,
		jobCap:        1,
		maxSize:       defaultMaxSize,
		maxTime:       1 * time.Second,
		useDocker:     true,
		expectJobFail: true,
	})

	// Test job should fail if it exceeds memory limit (docker only).
	runTest(t, testConfig{
		jobs:          1,
		workers:       1,
		jobCap:        1,
		maxSize:       defaultMaxSize,
		maxTime:       defaultMaxTime,
		memLimit:      5,
		useDocker:     true,
		expectJobFail: true,
	})
}

func TestJobCancel(t *testing.T) {
	d := NewDispatcher(1, 10)

	// Create five jobs.
	res1 := newMockResponseEventSink()
	job1 := NewJob(mockTestBody, res1, defaultMaxSize, defaultMaxTime, false, defaultMemLimit)

	res2 := newMockResponseEventSink()
	job2 := NewJob(mockTestBody, res2, defaultMaxSize, defaultMaxTime, false, defaultMemLimit)

	res3 := newMockResponseEventSink()
	job3 := NewJob(mockTestBody, res3, defaultMaxSize, defaultMaxTime, false, defaultMemLimit)

	res4 := newMockResponseEventSink()
	job4 := NewJob(mockTestBody, res4, defaultMaxSize, defaultMaxTime, false, defaultMemLimit)

	res5 := newMockResponseEventSink()
	job5 := NewJob(mockTestBody, res5, defaultMaxSize, defaultMaxTime, false, defaultMemLimit)

	// Cancel first job right away.
	job1.Cancel()

	// Queue all jobs.
	resultChan1, err := d.Enqueue(job1)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	resultChan2, err := d.Enqueue(job2)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	resultChan3, err := d.Enqueue(job3)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	resultChan4, err := d.Enqueue(job4)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	resultChan5, err := d.Enqueue(job5)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	r1 := <-resultChan1
	r2 := <-resultChan2

	// Cancel second job after it has already run.
	job2.Cancel()

	// Cancel fourth job after it has been queued, but before it has run.
	job4.Cancel()

	r3 := <-resultChan3
	r4 := <-resultChan4

	// Wait half a second to give job 5 a chance to start running, then cancel it.
	time.Sleep(500 * time.Millisecond)
	job5.Cancel()

	r5 := <-resultChan5

	// Check that jobs 1 and 4 failed.
	if r1.Success {
		t.Errorf("Expected job 1 to fail but it succeeded.")
	}
	if r4.Success {
		t.Errorf("Expected job 4 to fail but it succeeded.")
	}

	// Check that jobs 2, 3, and 5 succeeded.
	if !r2.Success {
		t.Errorf("Expected job 2 to succeed but it failed.")
	}
	if !r3.Success {
		t.Errorf("Expected job 3 to succeed but it failed.")
	}
	if !r5.Success {
		t.Errorf("Expected job 5 to succeed but it failed.")
	}
}
