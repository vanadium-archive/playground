// Functions to start services needed by the Vanadium playground.
// These should never trigger program exit.
// TODO(ivanpi): Use the modules library to start the services instead.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"time"

	"v.io/core/veyron/lib/flags/consts"

	"playground/lib"
)

var (
	proxyName = "proxy"
)

// TODO(ivanpi): this is all implemented in veyron/lib/modules/core, you
// may be able to use that directly.

// startMount starts a mounttabled process, and sets the NAMESPACE_ROOT env
// variable to the mounttable's location.  We run one mounttabled process for
// the entire environment.
func startMount(timeLimit time.Duration) (proc *os.Process, err error) {
	cmd := makeCmd("<mounttabled>", true, "mounttabled", "-veyron.tcp.address=127.0.0.1:0")
	matches, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile("NAME=(.*)"))
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
		"-log_dir=/tmp/logs",
		"-name="+proxyName,
		"-address=127.0.0.1:0",
		"-http=")
	if _, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile("NAME=(.*)")); err != nil {
		return nil, fmt.Errorf("Error starting proxy: %v", err)
	}
	return cmd.Process, nil
}

// startWspr starts a wsprd process. We run one wsprd process for each
// javascript file being run.
func startWspr(fileName, credentials string, timeLimit time.Duration) (proc *os.Process, port int, err error) {
	cmd := makeCmd("<wsprd>:"+fileName, true,
		"wsprd",
		"-veyron.proxy="+proxyName,
		"-veyron.tcp.address=127.0.0.1:0",
		"-port=0",
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
	parts, err := startAndWaitFor(cmd, timeLimit, regexp.MustCompile(".*port: (.*)"))
	if err != nil {
		return nil, 0, fmt.Errorf("Error starting wspr: %v", err)
	}
	portstr := parts[1]
	port, err = strconv.Atoi(portstr)
	if err != nil {
		return nil, 0, fmt.Errorf("Malformed port: %q: %v", portstr, err)
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
	cmd.Stdout.(*lib.MultiWriter).Add(writer)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan []string)
	go (func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if matches := outputRegexp.FindStringSubmatch(line); matches != nil {
				ch <- matches
			}
		}
		close(ch)
	})()
	select {
	case <-time.After(timeout):
		return nil, fmt.Errorf("Timeout starting service: %v", cmd.Path)
	case matches := <-ch:
		return matches, nil
	}
}
