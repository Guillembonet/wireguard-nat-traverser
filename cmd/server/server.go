package main

import (
	"fmt"
	"ias/project/communication"
	"net"
	"strings"
)

type server struct {
	client *communication.WireguardClient
}

const DEFAULT_DEVICE_NAME = "wg1"

func (s *server) handlePacket(buf []byte, rlen int, originAddr *net.UDPAddr, conn *net.UDPConn) error {
	message := strings.ReplaceAll(string(buf[0:rlen]), "\n", "")
	query := strings.Split(message, " ")
	if query[0] == "add" {
		communication.HandleAdd(query[1], query[2], s.client.AddPeer)
		//Reply with add
		ownPublicKey, err := s.client.GetDevicePublicKey()
		if err != nil {
			fmt.Printf("GetDevicePublicKey failed: %w\n", err)
			return err
		}
		interfaceIp, err := s.client.GetInterfaceIP()
		if err != nil {
			fmt.Printf("GetInterfaceIP failed: %w\n", err)
			return err
		}
		communication.SendUDPMessage(make([]byte, 1024), conn, fmt.Sprintf("add %s %s", *ownPublicKey, *interfaceIp), *originAddr, true)
	}
	if query[0] == "get" {
		publicKey := strings.ReplaceAll(message[4:], "\n", "")
		communication.SendUDPMessage(
			make([]byte, 1024),
			conn,
			fmt.Sprintf("{%s: \"%s\"}", publicKey, "hey"),
			*originAddr,
			true)
	}
	return nil
}

func main() {
	sock, err := communication.CreateUDPSocket(":2020")
	if err != nil {
		fmt.Printf("Failed: %w\n", err)
		return
	}
	defer sock.Close()

	wgClient, err := communication.NewWireguardClient(DEFAULT_DEVICE_NAME)
	if err != nil {
		fmt.Printf("New Wireguard client failed: %w\n", err)
		return
	}
	err = wgClient.ConfigureWireguardClient(2021)
	if err != nil {
		fmt.Printf("Configure wireguard client failed: %w\n", err)
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	err = wgClient.SetInterfaceIP("1")
	if err != nil {
		fmt.Printf("SetInterfaceIP failed: %w\n", err)
		return
	}

	server := &server{client: wgClient}

	for {
		buf := make([]byte, 1024)
		rlen, originAddr, err := sock.ReadFromUDP(buf)
		if err != nil {
			fmt.Println(err)
		}
		go server.handlePacket(buf, rlen, originAddr, sock)
	}
}
