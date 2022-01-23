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

	var cparams params.Client
	cparams.Init()

	flag.Parse()

	cparams.TunnelSlash24IP = gparams.TunnelSlash24IP

	errChan := make(chan error)

	client, err := container.ConstructClient(gparams, cparams)
	if err != nil {
		panic(err)
	}

	go func() {
		errChan <- client.Start()
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
