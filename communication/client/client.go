package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/guillembonet/wireguard-nat-traverser/communication"
	"github.com/guillembonet/wireguard-nat-traverser/connection"
	"github.com/guillembonet/wireguard-nat-traverser/constants"
	"github.com/guillembonet/wireguard-nat-traverser/params"
	"github.com/guillembonet/wireguard-nat-traverser/utils"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Client struct {
	manager connectionManager
	conn    *net.UDPConn
	params  params.Client
	server  *net.UDPAddr
}

type connectionManager interface {
	AddPeer(publicKey wgtypes.Key, cidr string, endpoint *net.UDPAddr, replacePeers bool) error
	GetPublicKey() (*wgtypes.Key, error)
	GetInterfaceIP() (string, error)
	SetInterfaceIP(ip string) error
}

func NewClient(manager connectionManager, conn *net.UDPConn, cparams params.Client) *Client {
	return &Client{
		manager: manager,
		conn:    conn,
		params:  cparams,
	}
}

func (c *Client) Start() error {
	go c.cli()
	return communication.ListenUDP(c.conn, c.handlePacket)
}

func (c *Client) handlePacket(message string, originAddr *net.UDPAddr) {
	args := utils.GetQuery(message)
	switch args[0] {
	// add <public_key> <ip>
	case constants.AddQuery:
		publicKey, err := wgtypes.ParseKey(args[1])
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		serverAddr, err := net.ResolveUDPAddr("udp", c.params.ServerIP+":"+strconv.Itoa(c.params.ServerWireguardPort))
		if err != nil {
			log.Println(fmt.Errorf("server address resolution failed: %w", err))
			return
		}
		err = c.manager.AddPeer(publicKey, args[2]+"/32", serverAddr, false)
		if err != nil {
			log.Println(fmt.Errorf("AddPeer failed: %w", err))
			return
		}
		// set server to vpn ip
		server, err := net.ResolveUDPAddr("udp", args[2]+":"+strconv.Itoa(c.params.ServerUDPPort))
		if err != nil {
			log.Println(fmt.Errorf("server tunnel address resolution failed: %w", err))
			return
		}
		c.server = server
		log.Printf("Server connection added. IP: %s\n", server.String())
		return
	// peer <peer_data>
	case constants.PeerQuery:
		peerData := &connection.Peer{}
		err := json.Unmarshal([]byte(args[1]), peerData)
		if err != nil {
			log.Println(fmt.Errorf("unmarshalling peer failed: %w", err))
			return
		}
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
		err = c.manager.AddPeer(publicKey, cidr, endpointAddr, false)
		if err != nil {
			log.Println(fmt.Errorf("AddPeer failed: %w", err))
			return
		}
		log.Printf("Added peer. Public key: %s. CIDR: %s. Endpoint: %s.\n", peerData.PublicKey, peerData.CIDR, peerData.Endpoint)
		return
	}
}

// func (c *Client) cli() {
// 	reader := bufio.NewReader(os.Stdin)
// 	msgBuf := make([]byte, 1024)

// 	for {
// 		text, _ := reader.ReadString('\n')
// 		query := utils.GetQuery(text)
// 		switch query[0] {
// 		//add <host_id>
// 		case "add":
// 			publicKey, err := c.client.GetDevicePublicKey()
// 			if err != nil {
// 				log.Println(fmt.Errorf("GetDevicePublicKey failed: %w", err))
// 				break
// 			}
// 			hostId := query[1]
// 			err = c.client.SetInterfaceIP(constants.DEFAULT_BASE_IP + hostId + "/24")
// 			if err != nil {
// 				log.Println(fmt.Errorf("SetInterfaceIP failed: %w", err))
// 				break
// 			}
// 			interfaceIp, err := c.client.GetInterfaceIP()
// 			if err != nil {
// 				log.Println(fmt.Errorf("GetInterfaceIP failed: %w", err))
// 				break
// 			}
// 			err = communication.SendUDPMessage(msgBuf, c.conn, fmt.Sprintf("add %s %s", *publicKey, *interfaceIp), *c.serverAddr, false)
// 			if err != nil {
// 				log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
// 			}
// 		//connect <public_key>
// 		case "connect":
// 			err := communication.SendUDPMessage(msgBuf, c.conn, "get "+query[1], *c.serverAddr, false)
// 			if err != nil {
// 				log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
// 			}
// 		// remove
// 		case "remove":
// 			c.client.RemovePeerByAllowedIP(c.serverAddr.String() + "/32")
// 			c.serverAddr = c.initialServerAddr
// 			log.Println("Removed server connection")
// 		// set consumer|provider
// 		case "set":
// 			if strings.HasPrefix(query[1], "c") {
// 				c.isConsumer = true
// 				log.Println("Consumer mode set")
// 				break
// 			}
// 			if strings.HasPrefix(query[1], "p") {
// 				c.isConsumer = false
// 				log.Println("Provider mode set")
// 			}
// 		// exit
// 		case "exit":
// 			err := communication.SendUDPMessage(msgBuf, c.conn, "exit", *c.serverAddr, false)
// 			if err != nil {
// 				log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
// 			}
// 			c.finished <- true
// 		}
// 	}
// }
