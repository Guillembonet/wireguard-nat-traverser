package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"ias/project/communication"
	"ias/project/utils"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const DEFAULT_DEVICE_NAME = "wg0"

type client struct {
	client     *communication.WireguardClient
	messages   chan *message
	stop       chan bool
	clientType string
}

type message struct {
	content    string
	rlen       int
	originAddr *net.UDPAddr
}

func (c *client) handlePacket(message string, originAddr *net.UDPAddr, conn *net.UDPConn) error {
	query := utils.GetQuery(message)
	originAddr.Port = 2021
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
		err = c.client.AddPeer(publicKey, ip, endpointAddr)
		if err != nil {
			return err
		}
		fmt.Printf("Added peer %s %s\n", query[1], ip)
	}
	if query[0] == "peer" {
		fmt.Println(query[1])
		peerData := &communication.PeerData{}
		json.Unmarshal([]byte(query[1]), peerData)
		publicKey, err := wgtypes.ParseKey(peerData.PublicKey)
		if err != nil {
			return err
		}
		endpointAddr, err := net.ResolveUDPAddr("udp", peerData.Endpoint)
		if err != nil {
			return err
		}
		cidr := peerData.Ip
		if strings.HasPrefix(c.clientType, "c") {
			cidr = "0.0.0.0/0"
		}
		err = c.client.AddPeer(publicKey, cidr, endpointAddr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) cli(conn *net.UDPConn, address *net.UDPAddr) {
	reader := bufio.NewReader(os.Stdin)
	msgBuf := make([]byte, 1024)

	for {
		text, _ := reader.ReadString('\n')
		query := utils.GetQuery(text)

		switch query[0] {
		case "add":
			publicKey, err := c.client.GetDevicePublicKey()
			if err != nil {
				fmt.Printf("GetDevicePublicKey failed: %w\n", err)
				return
			}
			hostId := query[1]
			err = c.client.SetInterfaceIP("10.0.0." + hostId)
			if err != nil {
				fmt.Printf("SetInterfaceIP failed: %w\n", err)
				return
			}
			interfaceIp, err := c.client.GetInterfaceIP()
			if err != nil {
				fmt.Printf("GetInterfaceIP failed: %w\n", err)
				return
			}
			communication.SendUDPMessage(msgBuf, conn, fmt.Sprintf("add %s %s", *publicKey, fmt.Sprintf("%s/32", *interfaceIp)), *address, true)
		case "connect":
			publicKey := query[1]
			communication.SendUDPMessage(msgBuf, conn, "get "+publicKey, *address, true)
		case "set":
			c.clientType = query[1]
		case "ping":
			address, _ := net.ResolveUDPAddr("udp", query[1])
			communication.SendUDPMessage(msgBuf, conn, "ping", *address, true)
		case "exit":
			communication.SendUDPMessage(msgBuf, conn, "exit", *address, true)
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
	sock, err := communication.CreateUDPSocket(":"+os.Args[5])
	if err != nil {
		fmt.Printf("Failed: %w\n", err)
		return
	}
	defer sock.Close()

	server, err := net.ResolveUDPAddr("udp", os.Args[1]+":"+os.Args[2])
	if err != nil {
		fmt.Printf("Could not resolve %s:%s\n", os.Args[1], os.Args[2])
		return
	}

	wgClient, err := communication.NewWireguardClient(os.Args[4])
	if err != nil {
		fmt.Printf("New Wireguard client failed: %w\n", err)
		return
	}
	port, err := strconv.Atoi(os.Args[6])
	if err != nil {
		fmt.Printf("No wireguard port supplied: %w\n", err)
		return
	}
	err = wgClient.ConfigureWireguardClient(port)
	if err != nil {
		fmt.Printf("Configure wireguard client failed: %w\n", err)
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	client := &client{client: wgClient, messages: make(chan *message), stop: make(chan bool)}
	go client.cli(sock, server)
	go client.handleMessages(sock)

mainLoop:
	for {
		select {
		case msg := <-client.messages:
			go client.handlePacket((*msg).content, (*msg).originAddr, sock)
		case stop := <-client.stop:
			if stop {
				break mainLoop
			}
		}
	}
}
