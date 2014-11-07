// +build OMIT
package main

import (
	"fmt"

	_ "veyron.io/veyron/veyron/profiles"
	"veyron.io/veyron/veyron2/rt"

	"pingpong"
)

func main() {
	runtime := rt.Init()
	log := runtime.Logger()

	s := pingpong.PingPongClient("pingpong")
	pong, err := s.Ping(runtime.NewContext(), "PING")
	if err != nil {
		log.Fatal("error pinging: ", err)
	}
	fmt.Println(pong)
}
