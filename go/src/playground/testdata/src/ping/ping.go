// +build ignore

package main

import (
	"fmt"

	"v.io/v23"
	"v.io/x/lib/vlog"
	_ "v.io/x/ref/profiles"

	"pingpong"
)

func main() {
	ctx, shutdown := v23.Init()
	defer shutdown()

	s := pingpong.PingPongClient("pingpong")

	fmt.Printf("Pinging\n")
	pong, err := s.Ping(ctx, "PING")
	if err != nil {
		vlog.Fatal("error pinging: ", err)
	}
	fmt.Println(pong)
}
