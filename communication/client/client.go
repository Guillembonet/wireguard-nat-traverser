package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"ias/project/communication"
	"ias/project/constants"
	"ias/project/utils"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Client struct {
	client              *communication.WireguardClient
	conn                *net.UDPConn
	msgBuf              []byte
	initialServerAddr   *net.UDPAddr
	serverAddr          *net.UDPAddr
	serverWireguardPort int
	finished            chan bool
	isConsumer          bool
}

func CreateClient(wgClient *communication.WireguardClient, conn *net.UDPConn, serverAddr *net.UDPAddr, serverWireguardPort int, finished chan bool) *Client {
	return &Client{
		client:              wgClient,
		conn:                conn,
		msgBuf:              make([]byte, 1024),
		initialServerAddr:   serverAddr,
		serverAddr:          serverAddr,
		serverWireguardPort: serverWireguardPort,
		finished:            finished,
		isConsumer:          false,
	}
}

func (c *Client) Start() {
	go communication.ListenUDP(c.conn, c.handlePacket)
	go c.cli()
}

func (c *Client) Close() error {
	err := c.conn.Close()
	if err != nil {
		return err
	}
	if c.isConsumer {
		err = communication.RemoveConsumerRules(c.client)
		if err != nil {
			log.Println(fmt.Errorf("RemoveConsumerRules failed: %w", err))
		}
	}
	return c.client.Close()
}

func (c *Client) handlePacket(message string, originAddr *net.UDPAddr) {
	query := utils.GetQuery(message)
	// add <public_key> <ip>
	if query[0] == "add" {
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		serverAddr, err := net.ResolveUDPAddr("udp", c.initialServerAddr.IP.String()+":"+strconv.Itoa(c.serverWireguardPort))
		if err != nil {
			log.Println(fmt.Errorf("server address resolution failed: %w", err))
			return
		}
		err = c.client.AddPeer(publicKey, query[2]+"/32", serverAddr, false)
		if err != nil {
			log.Println(fmt.Errorf("AddPeer failed: %w", err))
			return
		}
		// set server to vpn ip
		server, _ := net.ResolveUDPAddr("udp", query[2]+":"+strconv.Itoa(c.serverAddr.Port))
		c.serverAddr = server
		log.Printf("Server connection added. IP: %s\n", server.String())
	}
	// peer <peer_data>
	if query[0] == "peer" {
		peerData := &communication.PeerData{}
		json.Unmarshal([]byte(query[1]), peerData)
		publicKey, err := wgtypes.ParseKey(peerData.PublicKey)
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		endpointAddr, err := net.ResolveUDPAddr("udp", peerData.Endpoint)
		if err != nil {
			log.Println(fmt.Errorf("endpoint resolution failed: %w", err))
			return
		}
		cidr := peerData.CIDR
		if c.isConsumer {
			cidr = "0.0.0.0/0"
		}
		err = c.client.AddPeer(publicKey, cidr, endpointAddr, false)
		if err != nil {
			log.Println(fmt.Errorf("AddPeer failed: %w", err))
			return
		}
		if c.isConsumer {
			err = communication.CreateConsumerRules(c.client)
			if err != nil {
				log.Println(fmt.Errorf("CreateConsumerRules failed: %w", err))
				return
			}
		}
		log.Printf("Added peer. Public key: %s. CIDR: %s. Endpoint: %s.\n", peerData.PublicKey, peerData.CIDR, peerData.Endpoint)
	}
}

func (c *Client) cli() {
	reader := bufio.NewReader(os.Stdin)
	msgBuf := make([]byte, 1024)

	for {
		text, _ := reader.ReadString('\n')
		query := utils.GetQuery(text)
		switch query[0] {
		//add <host_id>
		case "add":
			publicKey, err := c.client.GetDevicePublicKey()
			if err != nil {
				log.Println(fmt.Errorf("GetDevicePublicKey failed: %w", err))
				break
			}
			hostId := query[1]
			err = c.client.SetInterfaceIP(constants.DEFAULT_BASE_IP + hostId + "/24")
			if err != nil {
				log.Println(fmt.Errorf("SetInterfaceIP failed: %w", err))
				break
			}
			interfaceIp, err := c.client.GetInterfaceIP()
			if err != nil {
				log.Println(fmt.Errorf("GetInterfaceIP failed: %w", err))
				break
			}
			err = communication.SendUDPMessage(msgBuf, c.conn, fmt.Sprintf("add %s %s", *publicKey, *interfaceIp), *c.serverAddr, false)
			if err != nil {
				log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			}
		//connect <public_key>
		case "connect":
			err := communication.SendUDPMessage(msgBuf, c.conn, "get "+query[1], *c.serverAddr, false)
			if err != nil {
				log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			}
		// remove
		case "remove":
			c.client.RemovePeerByAllowedIP(c.serverAddr.String() + "/32")
			c.serverAddr = c.initialServerAddr
			log.Println("Removed server connection")
		// set consumer|provider
		case "set":
			if strings.HasPrefix(query[1], "c") {
				c.isConsumer = true
				log.Println("Consumer mode set")
				break
			}
			if strings.HasPrefix(query[1], "p") {
				c.isConsumer = false
				log.Println("Provider mode set")
			}
		// exit
		case "exit":
			err := communication.SendUDPMessage(msgBuf, c.conn, "exit", *c.serverAddr, false)
			if err != nil {
				log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			}
			c.finished <- true
		}
	}
}
