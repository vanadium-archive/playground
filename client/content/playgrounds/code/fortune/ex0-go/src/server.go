// index=1
package main

import (
	"fmt"
	"log"
	"math/rand"

	"fortune"
	"veyron/lib/signals"
	vflag "veyron/security/flag"
	"veyron2/ipc"
	"veyron2/rt"
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
	// Create the runtime.
	r := rt.Init()

	// Create a new instance of the runtime's server functionality.
	s, err := r.NewServer()
	if err != nil {
		log.Fatal("failure creating server: ", err)
	}

	// Create the fortune server stub.
	serverFortune := fortune.NewServerFortune(newFortuned())

	// Create an endpoint and begin listening.
	if endpoint, err := s.Listen("tcp", "127.0.0.1:0"); err == nil {
		fmt.Printf("Listening at: %v\n", endpoint)
	} else {
		log.Fatal("error listening to service: ", err)
	}

	// Serve the fortune dispatcher at "fortune".
	if err := s.Serve("fortune", ipc.LeafDispatcher(serverFortune, vflag.NewAuthorizerOrDie())); err != nil {
		log.Fatal("error serving service: ", err)
	}

	// Wait forever.
	<-signals.ShutdownOnSignals()
}
