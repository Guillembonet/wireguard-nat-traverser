package main

import (
	"bufio"
	"fmt"
	"ias/project/communication"
	"net"
	"os"
	"strings"
	"sync"
)

const DEFAULT_DEVICE_NAME = "wg0"

type client struct {
	lock     sync.Mutex
	client   *communication.WireguardClient
	messages chan *message
	stop     chan bool
}

type message struct {
	content    string
	rlen       int
	originAddr *net.UDPAddr
}

func (c *client) handlePacket(message string, originAddr *net.UDPAddr, conn *net.UDPConn) {
	c.lock.Lock()
	query := strings.Split(strings.ReplaceAll(message, "\n", ""), " ")
	originAddr.Port = 2021
	if query[0] == "add" {
		err := communication.HandleAdd(query[1], query[2], c.client.AddPeer, originAddr)
		if err != nil {
			fmt.Printf("HandleAdd failed: %w\n", err)
			return
		}
	}
}

func (c *client) cli(conn *net.UDPConn, address *net.UDPAddr) {
	reader := bufio.NewReader(os.Stdin)
	msgBuf := make([]byte, 1024)

clifor:
	for {
		text, _ := reader.ReadString('\n')
		query := strings.Split(strings.ReplaceAll(text, "\n", ""), " ")

		switch query[0] {
		case "add":
			publicKey, err := c.client.GetDevicePublicKey()
			if err != nil {
				fmt.Printf("GetDevicePublicKey failed: %w\n", err)
				return
			}
			interfaceIp, err := c.client.GetInterfaceIP()
			if err != nil {
				fmt.Printf("GetInterfaceIP failed: %w\n", err)
				return
			}
			communication.SendUDPMessage(msgBuf, conn, fmt.Sprintf("add %s %s", *publicKey, fmt.Sprintf("%s/24", *interfaceIp)), *address, true)
		case "connect":
			fmt.Println(query[0])
			publicKey := query[1]
			hostId := query[2]
			err := c.client.SetInterfaceIP(hostId)
			if err != nil {
				fmt.Printf("SetInterfaceIP failed: %w\n", err)
				return
			}
			communication.SendUDPMessage(msgBuf, conn, "get "+publicKey, *address, true)
		case "exit":
			communication.SendUDPMessage(msgBuf, conn, "exit", *address, true)
			c.stop <- true
			break clifor
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
	sock, err := communication.CreateUDPSocket(":2000")
	if err != nil {
		fmt.Printf("Failed: %w\n", err)
		return
	}
	defer sock.Close()

	server, err := net.ResolveUDPAddr("udp", "192.168.1.23:2020")
	if err != nil {
		fmt.Printf("Could not resolve 127.0.0.1:2000\n")
		return
	}

	wgClient, err := communication.NewWireguardClient(DEFAULT_DEVICE_NAME)
	if err != nil {
		fmt.Printf("New Wireguard client failed: %w\n", err)
		return
	}
	err = wgClient.ConfigureWireguardClient(2001)
	if err != nil {
		fmt.Printf("Configure wireguard client failed: %w\n", err)
		wgClient.Close()
		return
	}
	defer wgClient.Close()

	wgClient.SetInterfaceIP("2")

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
