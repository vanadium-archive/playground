// +build OMIT
package main

import (
	"fmt"

	_ "v.io/core/veyron/profiles"
	"v.io/core/veyron2"

	"pingpong"
)

func main() {
	ctx, shutdown := veyron2.Init()
	defer shutdown()
	log := veyron2.GetLogger(ctx)

	s := pingpong.PingPongClient("pingpong")
	pong, err := s.Ping(ctx, "PING")
	if err != nil {
		log.Fatal("error pinging: ", err)
	}
	fmt.Println(pong)
}
