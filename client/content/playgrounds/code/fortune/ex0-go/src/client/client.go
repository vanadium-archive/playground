// index=3
package main

import (
	"fmt"
	"time"

	_ "v.io/core/veyron/profiles"
	"v.io/core/veyron2"

	"fortune"
)

func main() {
	// Create the Vanadium context.
	ctx, shutdown := veyron2.InitForTest()
	defer shutdown()

	// Create a new stub that binds to address without
	// using the name service.
	stub := fortune.FortuneClient("fortune")

	// Issue a Get() RPC.
	// We do this in a loop to give the server time to start up.
	var fortune string
	for {
		var err error
		if fortune, err = stub.Get(ctx); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println(fortune)
}
