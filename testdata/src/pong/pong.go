// +build OMIT
package main

import (
	"fmt"

	"v.io/core/veyron/lib/signals"
	"v.io/core/veyron/profiles"
	"v.io/core/veyron2"
	"v.io/core/veyron2/ipc"

	"pingpong"
)

type pongd struct{}

func (f *pongd) Ping(ctx ipc.ServerContext, message string) (result string, err error) {
	remote := ctx.RemoteBlessings().ForContext(ctx)
	fmt.Printf("%v: %q\n", remote, message)
	return "PONG", nil
}

func main() {
	ctx, shutdown := veyron2.InitForTest()
	defer shutdown()
	log := veyron2.GetLogger(ctx)

	s, err := veyron2.NewServer(ctx)
	if err != nil {
		log.Fatal("failure creating server: ", err)
	}
	log.Info("Waiting for ping")

	serverPong := pingpong.PingPongServer(&pongd{})

	if endpoint, err := s.Listen(profiles.LocalListenSpec); err == nil {
		fmt.Printf("Listening at: %v\n", endpoint)
	} else {
		log.Fatal("error listening to service: ", err)
	}

	if err := s.Serve("pingpong", serverPong, nil); err != nil {
		log.Fatal("error serving service: ", err)
	}

	// Wait forever.
	<-signals.ShutdownOnSignals(ctx)
}
