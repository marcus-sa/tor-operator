package tor_local_manager

import (
	"flag"
	"log"

	"k8s.io/client-go/rest"

	"github.com/marcus-sa/tor-operator/pkg/local"
)

// tor-manager main.
func main() {
	flag.Parse()

	//stopCh := signals.SetupSignalHandler()
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	localManager := local.New(config)
	err = localManager.Run()
	if err != nil {
		log.Fatalf("%v", err)
	}
}