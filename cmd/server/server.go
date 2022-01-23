package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/guillembonet/wireguard-nat-traverser/di"
	"github.com/guillembonet/wireguard-nat-traverser/params"
)

func main() {
	container := &di.Container{}
	defer container.Cleanup()

	var gparams params.Generic
	gparams.Init()

	flag.Parse()

	errChan := make(chan error)

	server, err := container.ConstructServer(gparams)
	if err != nil {
		panic(err)
	}

	go func() {
		errChan <- server.Start()
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	go func() {
		<-sigchan
		errChan <- fmt.Errorf("received an interrupt signal")
	}()

	err = <-errChan
	log.Println(err)
}
