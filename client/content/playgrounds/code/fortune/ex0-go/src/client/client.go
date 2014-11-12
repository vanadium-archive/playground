// index=3
package main

import (
	"fmt"
	"time"

	_ "veyron.io/veyron/veyron/profiles"
	"veyron.io/veyron/veyron2/rt"

	"fortune"
)

func main() {
	runtime := rt.Init()

	// Create a new stub that binds to address without
	// using the name service.
	s := fortune.FortuneClient("fortune")

	// Issue a Get() RPC.
	// We do this in a loop to give the server time to start up.
	var fortune string
	for {
		var err error
		if fortune, err = s.Get(runtime.NewContext()); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println(fortune)
}
