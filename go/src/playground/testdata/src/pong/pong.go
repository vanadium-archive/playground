// +build ignore

package main

import (
	"fmt"

	"v.io/core/veyron/lib/signals"
	_ "v.io/core/veyron/profiles"
	"v.io/core/veyron2"
	"v.io/core/veyron2/ipc"
	"v.io/core/veyron2/vlog"

	"pingpong"
)

type pongd struct{}

func (f *pongd) Ping(ctx ipc.ServerContext, message string) (result string, err error) {
	remote, _ := ctx.RemoteBlessings().ForContext(ctx)
	fmt.Printf("%v: %q\n", remote, message)
	return "PONG", nil
}

func main() {
	ctx, shutdown := veyron2.Init()
	defer shutdown()

	s, err := veyron2.NewServer(ctx)
	if err != nil {
		vlog.Fatal("failure creating server: ", err)
	}

	serverPong := pingpong.PingPongServer(&pongd{})

	fmt.Printf("Starting server\n")
	if endpoint, err := s.Listen(veyron2.GetListenSpec(ctx)); err == nil {
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
