// +build OMIT
package main

import (
	"fmt"

	_ "v.io/veyron/veyron/profiles"
	"v.io/veyron/veyron2/rt"

	"pingpong"
)

func main() {
	runtime, err := rt.New()
	if err != nil {
		panic(err)
	}
	defer runtime.Cleanup()

	log := runtime.Logger()

	s := pingpong.PingPongClient("pingpong")
	pong, err := s.Ping(runtime.NewContext(), "PING")
	if err != nil {
		log.Fatal("error pinging: ", err)
	}
	fmt.Println(pong)
}
