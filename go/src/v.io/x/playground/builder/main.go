// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Compiles and runs code for the Vanadium playground. Code is passed via
// os.Stdin as a JSON encoded request struct.

// NOTE(nlacasse): We use log.Panic() instead of log.Fatal() everywhere in this
// file.  We do this because log.Panic calls panic(), which allows any deferred
// function to run.  In particular, this will cause the mounttable and proxy
// processes to be killed in the event of a compilation error.  log.Fatal, on
// the other hand, calls os.Exit(1), which does not call deferred functions,
// and will leave proxy and mounttable processes running.  This is not a big
// deal for production environment, because the Docker instance gets cleaned up
// after each run, but during development and testing these extra processes can
// cause issues.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"v.io/x/playground/lib"
	"v.io/x/playground/lib/event"
	"v.io/x/ref/envvar"
)

var (
	verbose              = flag.Bool("verbose", true, "Whether to output debug messages.")
	includeServiceOutput = flag.Bool("includeServiceOutput", false, "Whether to stream service (mounttable, wspr, proxy) output to clients.")
	includeV23Env        = flag.Bool("includeV23Env", false, "Whether to log the output of \"v23 env\" before compilation.")
	// TODO(ivanpi): Separate out mounttable, proxy, wspr timeouts. Add compile timeout. Revise default.
	runTimeout = flag.Duration("runTimeout", 3*time.Second, "Time limit for running user code.")

	stopped = false    // Whether we have stopped execution of running files.
	out     event.Sink // Sink for writing events (debug and run output) to stdout as JSON, one event per line.
	mu      sync.Mutex
)

// Type of data sent to the builder on stdin.  Input should contain Files.  We
// look for a file whose Name ends with .id, and parse that into credentials.
//
// TODO(ribrdb): Consider moving credentials parsing into the http server.
type request struct {
	Files       []*codeFile
	Credentials []credentials
}

// Type of file data.  Only Name and Body should be initially set.  The other
// fields are added as the file is parsed.
type codeFile struct {
	Name string
	Body string
	// Language the file is written in.  Inferred from the file extension.
	lang string
	// Credentials to associate with the file's process.
	credentials string
	// The executable flag denotes whether the file should be executed as
	// part of the playground run. This is currently used only for
	// javascript files, and go files with package "main".
	executable bool
	// Name of the binary (for go files).
	binaryName string
	// Running cmd process for the file.
	cmd *exec.Cmd
	// Any subprocesses that are needed to support running the file (e.g. wspr).
	subprocs []*os.Process
	// The index of the file in the request.
	index int
}

type exit struct {
	name string
	err  error
}

func debug(args ...interface{}) {
	event.Debug(out, args...)
}

func panicOnError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func logV23Env() error {
	if *includeV23Env {
		return makeCmd("<environment>", false, "", "v23", "env").Run()
	}
	return nil
}

// All .go and .vdl files must have paths at least two directories deep,
// beginning with "src/".
//
// If no credentials are specified in the request, then all files will use the
// same principal.
func parseRequest(in io.Reader) (request, error) {
	debug("Parsing input")
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return request{}, err
	}
	var r request
	if err = json.Unmarshal(data, &r); err != nil {
		return r, err
	}
	m := make(map[string]*codeFile)
	for i := 0; i < len(r.Files); i++ {
		f := r.Files[i]
		f.index = i
		if path.Ext(f.Name) == ".id" {
			if len(r.Credentials) != 0 {
				return r, fmt.Errorf("multiple .id files provided")
			}
			if err := json.Unmarshal([]byte(f.Body), &r.Credentials); err != nil {
				return r, err
			}
			for _, c := range r.Credentials {
				if isReservedCredential(c.Name) {
					return r, fmt.Errorf("cannot use name %q, it is in the reserved set %v", c, reservedCredentials)
				}
			}
			r.Files = append(r.Files[:i], r.Files[i+1:]...)
			i--
		} else {
			switch path.Ext(f.Name) {
			case ".js":
				// JavaScript files are always executable.
				f.executable = true
				f.lang = "js"
			case ".go":
				// Go files will be marked as executable if their package name is
				// "main". This happens in the "maybeSetExecutableAndBinaryName"
				// function.
				f.lang = "go"
			case ".vdl":
				f.lang = "vdl"
			default:
				return r, fmt.Errorf("Unknown file type: %q", f.Name)
			}

			basename := path.Base(f.Name)
			if _, ok := m[basename]; ok {
				return r, fmt.Errorf("Two files with same basename: %q", basename)
			}
			m[basename] = f
		}
	}
	if len(r.Credentials) == 0 {
		// Run everything as the same principal.
		for _, file := range m {
			file.credentials = defaultCredentials
		}
		return r, nil
	}
	for _, creds := range r.Credentials {
		for _, basename := range creds.Files {
			// Check that the file associated with the credentials exists.  We ignore
			// cases where it doesn't because the test .id files get used for
			// multiple different code files.  See testdata/ids/authorized.id, for
			// example.
			if m[basename] != nil {
				m[basename].credentials = creds.Name
			}
		}
	}
	return r, nil
}

func writeFiles(files []*codeFile) error {
	debug("Writing files")
	for _, f := range files {
		if err := f.write(); err != nil {
			return fmt.Errorf("Error writing %q: %v", f.Name, err)
		}
	}
	return nil
}

// If compilation failed due to user error (bad input), returns badInput=true
// and cerr=nil. Only internal errors return non-nil cerr.
func compileFiles(files []*codeFile) (badInput bool, cerr error) {
	found := make(map[string]bool)
	for _, f := range files {
		found[f.lang] = true
	}
	if !found["go"] && !found["vdl"] {
		// No need to compile.
		return false, nil
	}

	debug("Compiling files")
	pwd, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("Error getting current directory: %v", err)
	}
	srcd := filepath.Join(pwd, "src")
	if err = os.Chdir(srcd); err != nil {
		panicOnError(out.Write(event.New("", "stderr", ".go or .vdl files outside src/ directory.")))
		return true, nil
	}
	os.Setenv("GOPATH", pwd+":"+os.Getenv("GOPATH"))
	os.Setenv("VDLPATH", pwd+":"+os.Getenv("VDLPATH"))
	// We set isService=false for compilation because "go install" only produces
	// output on error, and we always want clients to see such errors.
	// TODO(ivanpi): We assume *exec.ExitError results from uncompilable input
	// files; other cases can result from bugs in playground backend or compiler
	// itself.
	if found["js"] && found["vdl"] {
		debug("Generating VDL for Javascript")
		err = makeCmd("<compile>", false, "",
			"vdl", "generate", "-lang=Javascript", "-js-out-dir="+srcd, "./...").Run()
		if _, ok := err.(*exec.ExitError); ok {
			return true, nil
		} else if err != nil {
			return false, err
		}
	}
	if found["go"] {
		debug("Generating VDL for Go and compiling Go")
		err = makeCmd("<compile>", false, "",
			"v23", "go", "install", "./...").Run()
		if _, ok := err.(*exec.ExitError); ok {
			return true, nil
		} else if err != nil {
			return false, err
		}
	}
	if err = os.Chdir(pwd); err != nil {
		return false, fmt.Errorf("Error returning to parent directory: %v", err)
	}
	return false, nil
}

func runFiles(files []*codeFile) {
	debug("Running files")
	exit := make(chan exit)
	running := 0
	for _, f := range files {
		if f.executable {
			f.run(exit)
			running++
		}
	}

	timeout := time.After(*runTimeout)

	for running > 0 {
		select {
		case <-timeout:
			panicOnError(out.Write(event.New("", "stderr", "Ran for too long; terminated.")))
			stopAll(files)
		case status := <-exit:
			if status.err == nil {
				panicOnError(out.Write(event.New(status.name, "stdout", "Exited cleanly.")))
			} else {
				panicOnError(out.Write(event.New(status.name, "stderr", fmt.Sprintf("Exited with error: %v", status.err))))
			}
			running--
			stopAll(files)
		}
	}
}

func stopAll(files []*codeFile) {
	mu.Lock()
	defer mu.Unlock()
	if !stopped {
		stopped = true
		for _, f := range files {
			f.stop()
		}
	}
}

func (f *codeFile) maybeSetExecutableAndBinaryName() error {
	debug("Parsing package from", f.Name)
	file, err := parser.ParseFile(token.NewFileSet(), f.Name,
		strings.NewReader(f.Body), parser.PackageClauseOnly)
	if err != nil {
		return err
	}
	pkg := file.Name.String()
	if pkg == "main" {
		f.executable = true
		basename := path.Base(f.Name)
		f.binaryName = basename[:len(basename)-len(path.Ext(basename))]
	}
	return nil
}

func (f *codeFile) write() error {
	debug("Writing file", f.Name)
	if f.lang == "go" || f.lang == "vdl" {
		if err := f.maybeSetExecutableAndBinaryName(); err != nil {
			return err
		}
	}
	// Retain the original file tree structure.
	if err := os.MkdirAll(path.Dir(f.Name), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(f.Name, []byte(f.Body), 0644)
}

func (f *codeFile) startJs() error {
	wsprProc, wsprPort, err := startWspr(f.Name, f.credentials, *runTimeout)
	if err != nil {
		return fmt.Errorf("Error starting wspr: %v", err)
	}
	f.subprocs = append(f.subprocs, wsprProc)
	os.Setenv("WSPR", "http://localhost:"+strconv.Itoa(wsprPort))
	f.cmd = makeCmd(f.Name, false, "", "node", f.Name)
	return f.cmd.Start()
}

func (f *codeFile) startGo() error {
	f.cmd = makeCmd(f.Name, false, f.credentials, filepath.Join("bin", f.binaryName))
	return f.cmd.Start()
}

func (f *codeFile) run(ch chan exit) {
	debug("Running", f.Name)
	err := func() error {
		mu.Lock()
		defer mu.Unlock()
		if stopped {
			return fmt.Errorf("Execution has stopped; not running %q", f.Name)
		}

		switch f.lang {
		case "go":
			return f.startGo()
		case "js":
			return f.startJs()
		default:
			return fmt.Errorf("Cannot run file %q", f.Name)
		}
	}()
	if err != nil {
		debug("Failed to start", f.Name, "-", err)
		// Use a goroutine to avoid deadlock.
		go func() {
			ch <- exit{f.Name, err}
		}()
		return
	}

	// Wait for the process to exit and send result to channel.
	go func() {
		debug("Waiting for", f.Name)
		err := f.cmd.Wait()
		debug("Done waiting for", f.Name)
		ch <- exit{f.Name, err}
	}()
}

func (f *codeFile) stop() {
	debug("Attempting to stop", f.Name)
	if f.cmd == nil {
		debug("Cannot stop:", f.Name, "cmd is nil")
	} else if f.cmd.Process == nil {
		debug("Cannot stop:", f.Name, "cmd is not nil, but cmd.Process is nil")
	} else {
		debug("Sending SIGTERM to", f.Name)
		f.cmd.Process.Signal(syscall.SIGTERM)
	}
	for i, subproc := range f.subprocs {
		debug("Killing subprocess", i, "for", f.Name)
		subproc.Kill()
	}
}

// Creates a cmd whose outputs (stdout and stderr) are streamed to stdout as
// Event objects. If you want to watch the output streams yourself, add your
// own writer(s) to the MultiWriter before starting the command.
func makeCmd(fileName string, isService bool, credentials, progName string, args ...string) *exec.Cmd {
	cmd := exec.Command(progName, args...)
	cmd.Env = os.Environ()
	if credentials != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%s", envvar.Credentials, filepath.Join(credentialsDir, credentials)))
	}
	stdout, stderr := lib.NewMultiWriter(), lib.NewMultiWriter()
	prefix := ""
	if isService {
		prefix = "svc-"
	}
	if !isService || *includeServiceOutput {
		stdout.Add(event.NewStreamWriter(out, fileName, prefix+"stdout"))
		stderr.Add(event.NewStreamWriter(out, fileName, prefix+"stderr"))
	}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return cmd
}

func main() {
	// Remove any association with other credentials, start from a clean
	// slate.
	envvar.ClearCredentials()
	flag.Parse()

	out = event.NewJsonSink(os.Stdout, !*verbose)

	r, err := parseRequest(os.Stdin)
	panicOnError(err)

	// Create a common "identity provider" that will bless each principal
	// in this test (including mounttable, proxy, etc. that will be
	// started).
	//
	// TODO(ivanpi,ashankar): Credential management in this playground has
	// become very unwieldy. As of March 2015, ashankar@ just what was
	// expedient to get some other changes through, but what is left is
	// certainly hacky. If the plan is to move all this process management
	// to the "modules" framework, then we should be able to leverage that
	// to manage and set credentials for each subprocess. If not, will have
	// to think of something else, but in any case, should clean this up!
	panicOnError(createCredentials(
		append(baseCredentials(),
			rootCredentialsAtIdentityProvider(r.Credentials)...)))

	mt, err := startMount(*runTimeout)
	panicOnError(err)
	defer mt.Kill()

	proxy, err := startProxy(*runTimeout)
	panicOnError(err)
	defer proxy.Kill()

	panicOnError(writeFiles(r.Files))

	logV23Env()

	badInput, err := compileFiles(r.Files)
	// Panic on internal error, but not on user error.
	panicOnError(err)
	if badInput {
		panicOnError(out.Write(event.New("<compile>", "stderr", "Compilation error.")))
		return
	}
	runFiles(r.Files)
}
