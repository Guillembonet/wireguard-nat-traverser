package main

import (
	"encoding/json"
	"fmt"
	"ias/project/communication"
	"ias/project/utils"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type server struct {
	client *communication.WireguardClient
}

const DEFAULT_DEVICE_NAME = "wg1"

func (s *server) handlePacket(message string, originAddr *net.UDPAddr, conn *net.UDPConn) error {
	query := utils.GetQuery(message)
	if query[0] == "add" {
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			return err
		}
		ip := query[2]
		err = s.client.AddPeer(publicKey, ip, nil)
		if err != nil {
			return err
		}
		fmt.Printf("Added peer %s %s\n", query[1], ip)
		//Reply with add
		ownPublicKey, err := s.client.GetDevicePublicKey()
		if err != nil {
			return err
		}
		interfaceIp, err := s.client.GetInterfaceIP()
		if err != nil {
			return err
		}
		communication.SendUDPMessage(make([]byte, 1024), conn, fmt.Sprintf("add %s %s", *ownPublicKey, fmt.Sprintf("%s/32", *interfaceIp)), *originAddr, true)
	}
	if query[0] == "get" {
		//return peer data
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			return err
		}
		ip, endpoint, err := s.client.GetPeerIPAndEndpoint(publicKey)
		if err != nil {
			return err
		}
		data := &communication.PeerData{
			PublicKey: query[1],
			Ip:        ip,
			Endpoint:  endpoint,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		communication.SendUDPMessage(
			make([]byte, 1024),
			conn,
			fmt.Sprintf("peer %s", string(jsonData)),
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

	err = wgClient.SetInterfaceIP("10.0.0.1")
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
		go server.handlePacket(string(buf[0:rlen]), originAddr, sock)
	}
}
