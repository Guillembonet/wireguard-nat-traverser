package main

import (
	"fmt"
	"ias/project/communication"
	"ias/project/communication/client"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
)

const DEFAULT_DEVICE_NAME = "wg0"

func main() {
	if len(os.Args) < 7 {
		fmt.Println("Usage: sudo ./client <server_ip> <server_udp_port> <server_wireguard_port> <iface_name> <udp_port> <wireguard_port>")
		return
	}
	serverIP := os.Args[1]
	serverUDPPort := os.Args[2]
	serverWireguardPort, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Println(fmt.Errorf("no server wireguard port supplied or invalid: %w", err))
		return
	}
	interfaceName := os.Args[4]
	udpPort := os.Args[5]
	wireguardPort, err := strconv.Atoi(os.Args[6])
	if err != nil {
		log.Println(fmt.Errorf("no wireguard port supplied or invalid: %w", err))
		return
	}
	sock, err := communication.CreateUDPSocket(":" + udpPort)
	if err != nil {
		log.Println(fmt.Errorf("failed creating socket in port %s: %w", udpPort, err))
		sock.Close()
		return
	}
	defer sock.Close()

	server, err := net.ResolveUDPAddr("udp", serverIP+":"+serverUDPPort)
	if err != nil {
		log.Printf("Could not resolve %s:%s\n", serverIP, serverUDPPort)
		return
	}

	wgClient, err := communication.NewWireguardClient(interfaceName)
	if err != nil {
		log.Println(fmt.Errorf("new wireguard client failed: %w", err))
		return
	}
	err = wgClient.ConfigureWireguardClient(wireguardPort)
	if err != nil {
		log.Println(fmt.Errorf("configure wireguard client failed: %w", err))
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	finished := make(chan bool)

	publicKey, err := wgClient.GetDevicePublicKey()
	if err != nil {
		log.Println(fmt.Errorf("GetDevicePublicKey failed: %w", err))
		return
	}
	log.Printf("Your Public Key is %s\n", *publicKey)

	client := client.CreateClient(wgClient, sock, server, serverWireguardPort, finished)
	go client.Start()
	defer client.Close()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	select {
	case <-sigchan:
		break
	case <-finished:
		break
	}
	err = client.Close()
	if err != nil {
		log.Println(fmt.Errorf("unable to close client: %w", err))
	}
	log.Println("Cleaning up and closing...")
}
