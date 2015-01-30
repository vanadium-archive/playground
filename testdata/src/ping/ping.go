// +build ignore

package main

import (
	"fmt"

	_ "v.io/core/veyron/profiles"
	"v.io/core/veyron2"
	"v.io/core/veyron2/vlog"

	"pingpong"
)

func main() {
	ctx, shutdown := veyron2.Init()
	defer shutdown()

	s := pingpong.PingPongClient("pingpong")

	fmt.Printf("Pinging\n")
	pong, err := s.Ping(ctx, "PING")
	if err != nil {
		vlog.Fatal("error pinging: ", err)
	}
	fmt.Println(pong)
}
