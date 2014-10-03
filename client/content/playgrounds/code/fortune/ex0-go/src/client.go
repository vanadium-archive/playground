// index=3
package main

import (
	"fmt"
	"time"

	"fortune"
	"veyron2/rt"
)

func main() {
	runtime := rt.Init()
	log := runtime.Logger()

	// Create a new stub that binds to address without
	// using the name service.
	s, err := fortune.BindFortune("fortune")
	if err != nil {
		log.Fatal("error binding to server: ", err)
	}

	// Issue a Get() RPC.
	// We do this in a loop to give the server time to start up.
	var fortune string
	for {
		var err error
		if fortune, err = s.Get(runtime.NewContext()); err == nil {
			break;
		}
		time.Sleep(100 * time.Millisecond)

	}
	fmt.Println(fortune)
}
