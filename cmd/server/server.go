package main

import (
	"encoding/json"
	"fmt"
	"ias/project/communication"
	"ias/project/utils"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type server struct {
	client *communication.WireguardClient
}

func (s *server) handlePacket(message string, originAddr *net.UDPAddr, conn *net.UDPConn) error {
	query := utils.GetQuery(message)
	if query[0] == "add" {
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			return err
		}
		ip := query[2] + "/32"
		err = s.client.AddPeer(publicKey, ip, nil, false)
		if err != nil {
			return err
		}
		log.Printf("Added peer %s %s\n", query[1], ip)
		//Reply with add
		ownPublicKey, err := s.client.GetDevicePublicKey()
		if err != nil {
			return err
		}
		interfaceIp, err := s.client.GetInterfaceIP()
		if err != nil {
			return err
		}
		communication.SendUDPMessage(make([]byte, 1024), conn, fmt.Sprintf("add %s %s", *ownPublicKey, *interfaceIp), *originAddr, true)
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
	if query[0] == "remove" {
		cidr := originAddr.IP.String() + "/32"
		s.client.RemovePeerByAllowedIP(cidr)
		log.Printf("Removed %s\n", cidr)
	}
	return nil
}

func (s *server) listenUDP(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		rlen, originAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		go s.handlePacket(string(buf[0:rlen]), originAddr, conn)
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: sudo ./server <udp_port> <wireguard_port> <iface_name>")
		return
	}
	sock, err := communication.CreateUDPSocket(":" + os.Args[1])
	if err != nil {
		log.Printf("Failed: %w\n", err)
		return
	}
	defer sock.Close()

	wgClient, err := communication.NewWireguardClient(os.Args[3])
	if err != nil {
		log.Printf("New Wireguard client failed: %w\n", err)
		return
	}
	wireguardPort, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Printf("<wireguard_port> must be a number: %w\n", err)
		return
	}
	err = wgClient.ConfigureWireguardClient(wireguardPort)
	if err != nil {
		log.Printf("Configure wireguard client failed: %w\n", err)
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	err = wgClient.SetInterfaceIP("10.0.0.1")
	if err != nil {
		log.Printf("SetInterfaceIP failed: %w\n", err)
		return
	}

	server := &server{client: wgClient}
	go server.listenUDP(sock)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	log.Println("Cleaning up...")
	go server.client.Close()
	log.Println("Closing...")
}
