// +build ignore

// index=1
package main

import (
	"fmt"
	"math/rand"

	"v.io/v23"
	"v.io/v23/ipc"
	"v.io/x/lib/vlog"
	"v.io/x/ref/lib/signals"
	_ "v.io/x/ref/profiles"
	vflag "v.io/x/ref/security/flag"

	"fortune"
)

// The Fortuned implementation.
type fortuned struct {
	// The set of all fortunes.
	fortunes []string

	// Used to pick a random index in 'fortunes'.
	random *rand.Rand
}

// Initialize server state.
func newFortuned() *fortuned {
	return &fortuned{
		fortunes: []string{
			"You will reach the height of success in whatever you do.",
			"You have remarkable power which you are not using.",
			"Everything will now come your way.",
		},
		random: rand.New(rand.NewSource(99)),
	}
}

// Methods that get called by RPC requests.
func (f *fortuned) GetRandomFortune(_ ipc.ServerCall) (Fortune string, err error) {
	Fortune = f.fortunes[f.random.Intn(len(f.fortunes))]
	fmt.Printf("Serving: %s\n", Fortune)
	return Fortune, nil
}

func (f *fortuned) AddNewFortune(_ ipc.ServerCall, Fortune string) error {
	fmt.Printf("Adding: %s\n", Fortune)
	f.fortunes = append(f.fortunes, Fortune)
	return nil
}

// Main - Set everything up.
func main() {
	// Initialize Vanadium.
	ctx, shutdown := v23.Init()
	defer shutdown()

	// Create a new instance of the runtime's server functionality.
	server, err := v23.NewServer(ctx)
	if err != nil {
		vlog.Panic("failure creating server: ", err)
	}

	// Create the fortune server stub.
	fortuneServer := fortune.FortuneServer(newFortuned())

	// Create an endpoint and begin listening.
	if endpoint, err := server.Listen(v23.GetListenSpec(ctx)); err == nil {
		fmt.Printf("Listening at: %v\n", endpoint)
	} else {
		vlog.Panic("error listening at endpoint: ", err)
	}

	// Start the fortune server at "fortune".
	if err := server.Serve("bakery/cookie/fortune", fortuneServer, vflag.NewAuthorizerOrDie()); err == nil {
		fmt.Printf("Fortune server serving under: bakery/cookie/fortune\n")
	} else {
		vlog.Panic("error serving fortune server: ", err)
	}

	// Wait forever.
	<-signals.ShutdownOnSignals(ctx)
}