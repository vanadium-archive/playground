// +build OMIT
package main

import (
	"fmt"

	"veyron.io/veyron/veyron/lib/signals"
	"veyron.io/veyron/veyron/profiles"
	"veyron.io/veyron/veyron2/ipc"
	"veyron.io/veyron/veyron2/rt"

	"pingpong"
)

type pongd struct{}

func (f *pongd) Ping(ctx ipc.ServerContext, message string) (result string, err error) {
	remote := ctx.RemoteBlessings().ForContext(ctx)
	fmt.Printf("%v: %q\n", remote, message)
	return "PONG", nil
}

func main() {
	r := rt.Init()
	defer r.Cleanup()

	log := r.Logger()
	s, err := r.NewServer()
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
	<-signals.ShutdownOnSignals(r)
}
