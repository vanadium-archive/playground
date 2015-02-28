// +build ignore

package main

import (
	"fmt"

	"v.io/v23"
	"v.io/v23/ipc"
	"v.io/x/lib/vlog"
	"v.io/x/ref/lib/signals"
	_ "v.io/x/ref/profiles"

	"pingpong"
)

type pongd struct{}

func (f *pongd) Ping(ctx ipc.ServerCall, message string) (result string, err error) {
	remote, _ := ctx.RemoteBlessings().ForCall(ctx)
	fmt.Printf("%v: %q\n", remote, message)
	return "PONG", nil
}

func main() {
	ctx, shutdown := v23.Init()
	defer shutdown()

	s, err := v23.NewServer(ctx)
	if err != nil {
		vlog.Fatal("failure creating server: ", err)
	}

	serverPong := pingpong.PingPongServer(&pongd{})

	fmt.Printf("Starting server\n")
	if endpoint, err := s.Listen(v23.GetListenSpec(ctx)); err == nil {
		fmt.Printf("Listening at: %v\n", endpoint)
	} else {
		vlog.Fatal("error listening to service: ", err)
	}

	if err := s.Serve("pingpong", serverPong, nil); err == nil {
		fmt.Printf("Serving pingpong\n")
	} else {
		vlog.Fatal("error serving service: ", err)
	}

	// Wait forever.
	<-signals.ShutdownOnSignals(ctx)
}
