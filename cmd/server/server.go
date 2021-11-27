package main

import (
	"fmt"
	"ias/project/communication"
	"ias/project/communication/server"
	"log"
	"os"
	"os/signal"
	"strconv"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: sudo ./server <udp_port> <wireguard_port> <iface_name>")
		return
	}
	sock, err := communication.CreateUDPSocket(":" + os.Args[1])
	if err != nil {
		log.Println(fmt.Errorf("failed creating socket in port %s: %w", os.Args[1], err))
		sock.Close()
		return
	}
	defer sock.Close()

	wgClient, err := communication.NewWireguardClient(os.Args[3])
	if err != nil {
		log.Println(fmt.Errorf("new wireguard client failed: %w", err))
		return
	}
	wireguardPort, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Println(fmt.Errorf("<wireguard_port> must be a number: %w", err))
		return
	}
	err = wgClient.ConfigureWireguardClient(wireguardPort)
	if err != nil {
		log.Println(fmt.Errorf("configure wireguard client failed: %w", err))
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	err = wgClient.SetInterfaceIP("10.1.0.1")
	if err != nil {
		log.Println(fmt.Errorf("SetInterfaceIP failed: %w", err))
		return
	}

	server := server.CreateServer(wgClient, sock)
	go server.Start()
	defer server.Close()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	log.Println("Cleaning up and closing...")
}
