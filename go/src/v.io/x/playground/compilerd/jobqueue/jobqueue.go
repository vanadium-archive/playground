// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jobqueue implements a job queue for jobs.
//
// Usage:
//
//   dispatcher := NewDispatcher(numWorkers, maxWaitingJobs)
//
//   job := NewJob(...)
//   resultChan := dispatcher.Enqueue(job)
//   r :=<- resultChan // r will contain the result of running the job.
//
//   dispatcher.Stop() // Waits for any in-progress jobs to finish, cancels any
//                     // remaining jobs.
//
// Internally, the dispatcher has a channel of workers that represents a worker
// queue, and a channel of jobs that represents a job queue.  The dispatcher
// reads a  worker off the worker queue, and then reads a job off the job
// queue, and runs that job on that worker. When the job finishes, the worker
// is pushed back on to the worker queue.
//
// TODO(nlacasse): There are many types and functions exported in this file
// which are only exported because they are used by the compile test, in
// particular Job, Dispatcher, and Result types, and their constructors and
// methods.  Consider refactoring the compiler so those tests and the logic
// they test become part of this package.

package jobqueue

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	"v.io/x/playground/lib"
	"v.io/x/playground/lib/event"
	"v.io/x/playground/lib/log"
)

var (
	// A channel which returns unique playground ids.
	uniq = make(chan string)
)

func init() {
	val := time.Now().UnixNano()
	go func() {
		for {
			uniq <- fmt.Sprintf("playground_%d", val)
			val++
		}
	}()
}

type Job struct {
	id         string
	body       []byte
	res        *event.ResponseEventSink
	resultChan chan Result

	maxSize   int
	maxTime   time.Duration
	useDocker bool
	dockerMem int

	mu        sync.Mutex
	cancelled bool
}

func NewJob(body []byte, res *event.ResponseEventSink, maxSize int, maxTime time.Duration, useDocker bool, dockerMem int) *Job {
	return &Job{
		id:        <-uniq,
		body:      body,
		res:       res,
		maxSize:   maxSize,
		maxTime:   maxTime,
		useDocker: useDocker,
		dockerMem: dockerMem,

		// resultChan has capacity 1 so that writing to the channel won't block
		// if nobody ever reads the result.
		resultChan: make(chan Result, 1),
	}
}

// Body is a getter for Job.body.
func (j *Job) Body() []byte {
	return j.body
}

// Cancel will prevent the job from being run, if it has not already been
// started by a worker.
func (j *Job) Cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	log.Debugf("Cancelling job %v.", j.id)
	j.cancelled = true
}

// Dispatcher is an interface type so it can be mocked during tests.
type Dispatcher interface {
	Enqueue(j *Job) (chan Result, error)
	Stop()
}

// dispatcherImpl implements Dispatcher interface.
type dispatcherImpl struct {
	jobQueue chan *Job

	// A message sent on the stopped channel causes the dispatcher to stop
	// assigning new jobs to workers.
	stopped chan bool

	// wg represents currently running workers. It is used during Stop to make
	// sure that all workers have finished running their active jobs.
	wg sync.WaitGroup
}

var _ = Dispatcher((*dispatcherImpl)(nil))

func NewDispatcher(workers int, jobQueueCap int) Dispatcher {
	log.Debugf("Creating new dispatcher with %v workers and %v queue capacity.", workers, jobQueueCap)
	d := &dispatcherImpl{
		jobQueue: make(chan *Job, jobQueueCap),
		stopped:  make(chan bool),
	}

	d.start(workers)
	return d
}

// start starts a given number of workers, then reads from the jobQueue and
// assigns jobs to free workers.
func (d *dispatcherImpl) start(num int) {
	log.Debug("Dispatcher starting.")

	// Workers are published on the workerQueue when they are free.
	workerQueue := make(chan *worker, num)

	for i := 0; i < num; i++ {
		worker := newWorker(i)
		workerQueue <- worker
	}

	d.wg.Add(1)

	go func() {
	Loop:
		for {
			// Wait for the next available worker.
			select {
			case <-d.stopped:
				break Loop
			case worker := <-workerQueue:
				// Read the next job from the job queue.
				select {
				case <-d.stopped:
					break Loop
				case job := <-d.jobQueue:
					job.mu.Lock()
					cancelled := job.cancelled
					job.mu.Unlock()
					if cancelled {
						log.Debugf("Dispatcher encountered cancelled job %v, rejecting.", job.id)
						job.resultChan <- Result{
							Success: false,
							Events:  nil,
						}
						workerQueue <- worker
					} else {
						log.Debugf("Dispatching job %v to worker %v.", job.id, worker.id)
						d.wg.Add(1)
						go func() {
							job.resultChan <- worker.run(job)
							log.Debugf("Job %v finished on worker %v.", job.id, worker.id)
							d.wg.Done()
							workerQueue <- worker
						}()
					}
				}
			}
		}

		log.Debug("Dispatcher stopped.")

		// Dispatcher stopped, treat all remaining jobs as cancelled.
		for {
			select {
			case job := <-d.jobQueue:
				log.Debugf("Dispatcher is stopped, rejecting job %v.", job.id)
				job.resultChan <- Result{
					Success: false,
					Events:  nil,
				}
			default:
				log.Debug("Dispatcher job queue drained.")
				d.wg.Done()
				return
			}
		}
	}()
}

// Stop stops the dispatcher from assigning any new jobs to workers. Jobs that
// are currently running are allowed to continue. Other jobs are treated as
// cancelled. Stop blocks until all jobs have finished.
// TODO(nlacasse): Consider letting the dispatcher run all currently queued
// jobs, rather than rejecting them.  Or, put logic in the client to retry
// cancelled jobs.
func (d *dispatcherImpl) Stop() {
	log.Debug("Stopping dispatcher.")
	d.stopped <- true

	// Wait for workers to finish their current jobs.
	d.wg.Wait()
}

// Enqueue queues a job to be run be the next available worker. It returns a
// channel on which the job's results will be published.
func (d *dispatcherImpl) Enqueue(j *Job) (chan Result, error) {
	select {
	case d.jobQueue <- j:
		return j.resultChan, nil
	default:
		return nil, fmt.Errorf("Error queuing job. Job queue full.")
	}
}

type Result struct {
	Success bool
	Events  []event.Event
}

type worker struct {
	id int
}

func newWorker(id int) *worker {
	return &worker{
		id: id,
	}
}

// run compiles and runs a job, caches the result, and returns the result on
// the job's result channel.
func (w *worker) run(j *Job) Result {
	event.Debug(j.res, "Preparing to run program")

	memoryFlag := fmt.Sprintf("%dm", j.dockerMem)

	var cmd *exec.Cmd
	if j.useDocker {
		// TODO(nlacasse,ivanpi): Limit the CPU resources used by this docker
		// builder instance.  The docker "cpu-shares" flag can only limit on
		// docker process relative to another, so it's not useful for limiting
		// the cpu resources of all build instances. The docker "cpuset" flag
		// can pin the instance to a specific processor, so that might be of
		// use.
		cmd = docker("run", "-i",
			"--name", j.id,
			// Disable external networking.
			"--net", "none",
			// Limit instance memory.
			"--memory", memoryFlag,
			// Limit instance memory+swap combined.
			// Setting to the same value as memory effectively disables swap.
			"--memory-swap", memoryFlag,
			"playground")
	} else {
		// Run builder directly, without Docker. This should only happen during
		// development and in tests, never in production.
		cmd = exec.Command("builder")

		// Run the builder in a temp dir, so the bundle files and binaries do
		// not clutter up the current working dir. This also allows parallel
		// bundler runs, since otherwise the files from different runs stomp on
		// each other.
		tmpDir, err := ioutil.TempDir("", "pg-builder-")
		if err != nil {
			log.Panicf("Error creating temp dir for builder: %v\n", err)
		} else {
			cmd.Dir = tmpDir
		}
	}
	cmdKill := lib.DoOnce(func() {
		event.Debug(j.res, "Killing program")
		// The docker client can get in a state where stopping/killing/rm-ing
		// the container will not kill the client. The opposite should work
		// correctly (killing the docker client stops the container).
		// If not, the docker rm call below will.
		// Note, this wouldn't be sufficient if docker was called through sudo
		// since sudo doesn't pass sigkill to child processes.
		cmd.Process.Kill()
	})

	cmd.Stdin = bytes.NewReader(j.body)

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
		log.Warn(j.id, " builder stderr output too large, killing.")
		erroredOut = true
		cmdKill()
	}
	userErrorCallback := func(err error) {
		// A relay error can result from unparseable JSON caused by a builder bug
		// or a malicious exploit inside Docker. Panicking could lead to a DoS.
		log.Error(j.id, " builder stdout relay error: ", err)
		erroredOut = true
		cmdKill()
	}

	outRelay, outStop := event.LimitedEventRelay(j.res, j.maxSize, userLimitCallback, userErrorCallback)
	// Builder stdout should already contain a JSON Event stream.
	cmd.Stdout = outRelay

	// Any stderr is unexpected, most likely a bug (panic) in builder, but could
	// also result from a malicious exploit inside Docker.
	// It is quietly logged as long as it doesn't exceed maxSize.
	errBuffer := new(bytes.Buffer)
	cmd.Stderr = lib.NewLimitedWriter(errBuffer, j.maxSize, systemLimitCallback)

	event.Debug(j.res, "Running program")

	timeout := time.After(j.maxTime)
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

	event.Debug(j.res, "Program exited")

	// Return the appropriate error message to the client.
	if timedOut {
		j.res.Write(event.New("", "stderr", "Internal timeout, please retry."))
	} else if erroredOut {
		j.res.Write(event.New("", "stderr", "Internal error, please retry."))
	} else if sizedOut {
		j.res.Write(event.New("", "stderr", "Program output too large, killed."))
	}

	// Log builder internal errors, if any.
	// TODO(ivanpi): Prevent caching? Report to client if debug requested?
	if errBuffer.Len() > 0 {
		log.Warn(j.id, " builder stderr: ", errBuffer.String())
	}

	event.Debug(j.res, "Response finished")

	// TODO(nlacasse): This "docker rm" can be slow (several seconds), and seems
	// to block other Docker commands, thereby slowing down other concurrent
	// requests. We should figure out how to make it not block other Docker
	// commands. Setting GOMAXPROCS may or may not help.
	// See: https://github.com/docker/docker/issues/6480
	if j.useDocker {
		go func() {
			docker("rm", "-f", j.id).Run()
		}()
	} else {
		// Clean up after the builder process.
		os.RemoveAll(cmd.Dir)
	}

	// If we timed out or errored out, do not cache anything.
	// TODO(sadovsky): This policy is helpful for development, but may not be wise
	// for production. Revisit.
	if !timedOut && !erroredOut {
		return Result{
			Success: true,
			Events:  j.res.PopWrittenEvents(),
		}
	} else {
		return Result{
			Success: false,
			Events:  nil,
		}
	}
}

func docker(args ...string) *exec.Cmd {
	return exec.Command("docker", args...)
}
