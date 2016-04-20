// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// pg-index=2
package main

import (
	"fmt"
	"os"
	"time"

	"v.io/v23"
	"v.io/v23/context"
	_ "v.io/x/ref/runtime/factories/roaming"

	"fortune"
)

func main() {
	// Initialize Vanadium.
	ctx, shutdown, err := v23.TryInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize vanadium runtime: %v", err)
		os.Exit(1)
	}
	defer shutdown()

	// Create a new stub that binds to address without
	// using the name service.
	stub := fortune.FortuneClient("bakery/cookie/fortune")

	// Issue a Get() RPC.
	// We do this in a loop to give the server time to start up.
	fmt.Printf("Issuing request\n")
	var fortune string
	for {
		// Create a context that will timeout in 1 second.
		timeoutCtx, _ := context.WithTimeout(ctx, 1*time.Second)

		var err error
		if fortune, err = stub.GetRandomFortune(timeoutCtx); err == nil {
			break
		}
		fmt.Printf("%v\nRetrying in 100ms...\n", err)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("Received: %s\n", fortune)
}
