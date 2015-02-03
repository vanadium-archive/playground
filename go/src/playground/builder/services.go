// Functions to start services needed by the Vanadium playground.
// These should never trigger program exit.
// TODO(ivanpi): Use the modules library to start the services instead.

package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"v.io/core/veyron/lib/flags/consts"

	"playground/lib"
)

var (
	proxyName = "proxy"
)

// Note: This was copied from release/go/src/v.io/core/veyron/tools/findunusedport.
// I would like to be able to import that package directly, but it defines a
// main(), so can't be imported.  An alternative solution would be to call the
// 'findunusedport' binary, but that would require starting another process and
// parsing the output.  It seemed simpler to just copy the function here.
func findUnusedPort() (int, error) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 1000; i++ {
		port := 1024 + rnd.Int31n(64512)
		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
		if err != nil {
			continue
		}
		sa := &syscall.SockaddrInet4{Port: int(port)}
		if err := syscall.Bind(fd, sa); err != nil {
			continue
		}
		syscall.Close(fd)
		return int(port), nil
	}
	return 0, fmt.Errorf("Can't find unused port.")
}

// startMount starts a mounttabled process, and sets the NAMESPACE_ROOT env
// variable to the mounttable's location.  We run one mounttabled process for
// the entire environment.
func startMount(timeLimit time.Duration) (proc *os.Process, err error) {
	cmd := makeCmd("<mounttabled>", true, "mounttabled", "-veyron.tcp.address=127.0.0.1:0")
	matches, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile("Mount table .+ endpoint: (.+)\n"))
	if err != nil {
		return nil, fmt.Errorf("Error starting mounttabled: %v", err)
	}
	endpoint := matches[1]
	if endpoint == "" {
		return nil, fmt.Errorf("Failed to get mounttable endpoint")
	}
	return cmd.Process, os.Setenv(consts.NamespaceRootPrefix, endpoint)
}

// startProxy starts a proxyd process.  We run one proxyd process for the
// entire environment.
func startProxy(timeLimit time.Duration) (proc *os.Process, err error) {
	cmd := makeCmd("<proxyd>", true,
		"proxyd",
		// Verbose logging so we can watch the output for "Proxy listening" log line.
		"-v=1",
		"-name="+proxyName,
		"-address=127.0.0.1:0",
		"-http=")
	if _, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile("Proxy listening")); err != nil {
		return nil, fmt.Errorf("Error starting proxy: %v", err)
	}
	return cmd.Process, nil
}

// startWspr starts a wsprd process. We run one wsprd process for each
// javascript file being run.
func startWspr(fileName, credentials string, timeLimit time.Duration) (proc *os.Process, port int, err error) {
	port, err = findUnusedPort()
	if err != nil {
		return nil, port, err
	}
	cmd := makeCmd("<wsprd>:"+fileName, true,
		"wsprd",
		// Verbose logging so we can watch the output for "Listening" log line.
		"-v=1",
		"-veyron.proxy="+proxyName,
		"-veyron.tcp.address=127.0.0.1:0",
		"-port="+strconv.Itoa(port),
		// Retry RPC calls for 1 second. If a client makes an RPC call before the
		// server is running, it won't immediately fail, but will retry while the
		// server is starting.
		// TODO(nlacasse): Remove this when javascript can tell wspr how long to
		// retry for. Right now it's a global setting in wspr.
		"-retry-timeout=1",
		// The identd server won't be used, so pass a fake name.
		"-identd=/unused")
	if credentials != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%s", consts.VeyronCredentials, path.Join("credentials", credentials)))
	}
	if _, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile("Listening")); err != nil {
		return nil, 0, fmt.Errorf("Error starting wspr: %v", err)
	}
	return cmd.Process, port, nil
}

// Helper function to start a command and wait for output.  Arguments are a cmd
// to run, a timeout, and a regexp.  The slice of strings matched by the regexp
// is returned.
// TODO(nlacasse): Consider standardizing how services log when they start
// listening, and their endpoints (if any).  Then this could become a common
// util function.
func startAndWaitFor(cmd *exec.Cmd, timeout time.Duration, outputRegexp *regexp.Regexp) ([]string, error) {
	reader, writer := io.Pipe()
	// TODO(sadovsky): Why must we listen to both stdout and stderr? We should
	// know which one produces the "Listening" log line...
	cmd.Stdout.(*lib.MultiWriter).Add(writer)
	cmd.Stderr.(*lib.MultiWriter).Add(writer)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	buf := bufio.NewReader(reader)
	t := time.After(timeout)
	ch := make(chan []string)
	go (func() {
		for line, err := buf.ReadString('\n'); err == nil; line, err = buf.ReadString('\n') {
			if matches := outputRegexp.FindStringSubmatch(line); matches != nil {
				ch <- matches
			}
		}
		close(ch)
	})()
	select {
	case <-t:
		return nil, fmt.Errorf("Timeout starting service.")
	case matches := <-ch:
		return matches, nil
	}
}
