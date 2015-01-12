// index=1
package main

import (
	"fmt"
	"math/rand"

	"v.io/core/veyron/lib/signals"
	"v.io/core/veyron/profiles"
	vflag "v.io/core/veyron/security/flag"
	"v.io/core/veyron2"
	"v.io/core/veyron2/ipc"
	"v.io/core/veyron2/rt"

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
func (f *fortuned) Get(_ ipc.ServerContext) (Fortune string, err error) {
	return f.fortunes[f.random.Intn(len(f.fortunes))], nil
}

func (f *fortuned) Add(_ ipc.ServerContext, Fortune string) error {
	f.fortunes = append(f.fortunes, Fortune)
	return nil
}

// Main - Set everything up.
func main() {
	// Create the runtime and context.
	runtime, err := rt.New()
	if err != nil {
		panic(err)
	}
	defer runtime.Cleanup()
	ctx := runtime.NewContext()
	log := veyron2.GetLogger(ctx)

	// Create a new instance of the runtime's server functionality.
	server, err := veyron2.NewServer(ctx)
	if err != nil {
		log.Panic("failure creating server: ", err)
	}

	// Create the fortune server stub.
	fortuneServer := fortune.FortuneServer(newFortuned())

	// Create an endpoint and begin listening.
	if endpoint, err := server.Listen(profiles.LocalListenSpec); err == nil {
		fmt.Printf("Listening at: %v\n", endpoint)
	} else {
		log.Panic("error listening to service: ", err)
	}

	// Start the fortune server at "fortune".
	if err := server.Serve("fortune", fortuneServer, vflag.NewAuthorizerOrDie()); err != nil {
		log.Panic("error serving service: ", err)
	}

	// Wait forever.
	<-signals.ShutdownOnSignals(ctx)
}
