package main

import (
	"bufio"
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

const DEFAULT_DEVICE_NAME = "wg0"

type client struct {
	client   *communication.WireguardClient
	messages chan *message
	stop     chan bool
	server   *net.UDPAddr
}

type message struct {
	content    string
	rlen       int
	originAddr *net.UDPAddr
}

func (c *client) handlePacket(message string, originAddr *net.UDPAddr, conn *net.UDPConn) error {
	query := utils.GetQuery(message)
	if query[0] == "add" {
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			return err
		}
		ip := query[2]
		endpointAddr, err := net.ResolveUDPAddr("udp", os.Args[1]+":"+os.Args[3])
		if err != nil {
			return err
		}
		err = c.client.AddPeer(publicKey, ip+"/32", endpointAddr, false)
		if err != nil {
			return err
		}
		log.Printf("Added peer %s %s\n", query[1], ip+"/32")
		server, _ := net.ResolveUDPAddr("udp", ip+":"+os.Args[2])
		c.server = server
		log.Printf("Server connection encrypted wg ip: %s\n", server.String())
	}
	if query[0] == "peer" {
		peerData := &communication.PeerData{}
		json.Unmarshal([]byte(query[1]), peerData)
		log.Printf("Adding peer %s...\n", peerData.PublicKey)
		publicKey, err := wgtypes.ParseKey(peerData.PublicKey)
		if err != nil {
			return err
		}
		endpointAddr, err := net.ResolveUDPAddr("udp", peerData.Endpoint)
		if err != nil {
			return err
		}
		cidr := peerData.Ip
		err = c.client.AddPeer(publicKey, cidr, endpointAddr, false)
		if err != nil {
			return err
		}
		log.Printf("Added peer %s %s %s\n", peerData.PublicKey, peerData.Ip, peerData.Endpoint)
	}
	return nil
}

func (c *client) cli(conn *net.UDPConn) {
	reader := bufio.NewReader(os.Stdin)
	msgBuf := make([]byte, 1024)

	for {
		text, _ := reader.ReadString('\n')
		query := utils.GetQuery(text)
		switch query[0] {
		case "add":
			publicKey, err := c.client.GetDevicePublicKey()
			if err != nil {
				log.Printf("GetDevicePublicKey failed: %w\n", err)
				return
			}
			hostId := query[1]
			err = c.client.SetInterfaceIP("10.0.0." + hostId)
			if err != nil {
				log.Printf("SetInterfaceIP failed: %w\n", err)
				return
			}
			interfaceIp, err := c.client.GetInterfaceIP()
			if err != nil {
				log.Printf("GetInterfaceIP failed: %w\n", err)
				return
			}
			communication.SendUDPMessage(msgBuf, conn, fmt.Sprintf("add %s %s", *publicKey, *interfaceIp), *c.server, true)
		case "connect":
			publicKey := query[1]
			communication.SendUDPMessage(msgBuf, conn, "get "+publicKey, *c.server, true)
		case "remove":
			publicKey, err := c.client.GetDevicePublicKey()
			if err != nil {
				log.Printf("GetDevicePublicKey failed: %w\n", err)
				return
			}
			communication.SendUDPMessage(msgBuf, conn, "remove "+*publicKey, *c.server, true)
			c.client.RemovePeerByAllowedIP("10.0.0.1/32")
			server, err := net.ResolveUDPAddr("udp", os.Args[1]+":"+os.Args[2])
			if err != nil {
				log.Printf("Could not resolve %s:%s\n", os.Args[1], os.Args[2])
				return
			}
			c.server = server
			log.Println("Removed server connection")
		case "exit":
			communication.SendUDPMessage(msgBuf, conn, "exit", *c.server, true)
			c.stop <- true
		}
	}
}

func (c *client) handleMessages(conn *net.UDPConn) error {
	buf := make([]byte, 1024)
	for {
		rlen, originAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		c.messages <- &message{content: string(buf[0:rlen]), rlen: rlen, originAddr: originAddr}
	}
}

func main() {
	if len(os.Args) < 7 {
		fmt.Println("Usage: sudo ./client <server_ip> <server_udp_port> <server_wireguard_port> <iface_name> <udp_port> <wireguard_port>")
		return
	}
	sock, err := communication.CreateUDPSocket(":" + os.Args[5])
	if err != nil {
		log.Printf("Failed: %w\n", err)
		return
	}
	defer sock.Close()

	server, err := net.ResolveUDPAddr("udp", os.Args[1]+":"+os.Args[2])
	if err != nil {
		log.Printf("Could not resolve %s:%s\n", os.Args[1], os.Args[2])
		return
	}

	wgClient, err := communication.NewWireguardClient(os.Args[4])
	if err != nil {
		log.Printf("New Wireguard client failed: %w\n", err)
		return
	}
	port, err := strconv.Atoi(os.Args[6])
	if err != nil {
		log.Printf("No wireguard port supplied: %w\n", err)
		return
	}
	err = wgClient.ConfigureWireguardClient(port)
	if err != nil {
		log.Printf("Configure wireguard client failed: %w\n", err)
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	client := &client{client: wgClient, messages: make(chan *message), stop: make(chan bool), server: server}
	go client.cli(sock)
	go client.handleMessages(sock)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

mainLoop:
	for {
		select {
		case msg := <-client.messages:
			go client.handlePacket((*msg).content, (*msg).originAddr, sock)
		case stop := <-client.stop:
			if stop {
				break mainLoop
			}
		case <-sigchan:
			log.Println("Cleaning up...")
			go client.client.Close()
			log.Println("Closing...")
			break mainLoop
		}
	}
}
