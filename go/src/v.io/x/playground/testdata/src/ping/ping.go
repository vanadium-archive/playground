// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"fmt"

	"v.io/x/lib/vlog"
	_ "v.io/x/ref/runtime/factories/roaming"
	"v.io/x/ref/test"

	"pingpong"
)

func main() {
	ctx, shutdown := test.V23Init()
	defer shutdown()

	s := pingpong.PingPongClient("pingpong")

	fmt.Printf("Pinging\n")
	pong, err := s.Ping(ctx, "PING")
	if err != nil {
		vlog.Fatal("error pinging: ", err)
	}
	fmt.Println(pong)
}
