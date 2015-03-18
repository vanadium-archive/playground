// +build ignore

// index=2
package main

import (
	"fmt"
	"time"

	"v.io/v23"
	_ "v.io/x/ref/profiles"

	"fortune"
)

func main() {
	// Initialize Vanadium.
	ctx, shutdown := v23.Init()
	defer shutdown()

	// Create a new stub that binds to address without
	// using the name service.
	stub := fortune.FortuneClient("bakery/cookie/fortune")

	// Issue a Get() RPC.
	// We do this in a loop to give the server time to start up.
	fmt.Printf("Issuing request\n")
	var fortune string
	for {
		var err error
		if fortune, err = stub.GetRandomFortune(ctx); err == nil {
			break
		}
		fmt.Printf("%v\nRetrying in 100ms...\n", err)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("Received: %s\n", fortune)
}
