// +build ignore

// index=2
package main

import (
	"fmt"
	"time"

	_ "v.io/core/veyron/profiles"
	"v.io/core/veyron2"

	"fortune"
)

func main() {
	// Initialize Vanadium.
	ctx, shutdown := veyron2.Init()
	defer shutdown()

	// Create a new stub that binds to address without
	// using the name service.
	stub := fortune.FortuneClient("fortune")

	// Issue a Get() RPC.
	// We do this in a loop to give the server time to start up.
	fmt.Printf("Issuing request\n")
	var fortune string
	for {
		var err error
		if fortune, err = stub.Get(ctx); err == nil {
			break
		}
		fmt.Printf("%v\nRetrying in 100ms...\n", err)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("Received: %s\n", fortune)
}
