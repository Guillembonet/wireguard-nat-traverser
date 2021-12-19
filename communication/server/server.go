package server

import (
	"encoding/json"
	"fmt"
	"ias/project/communication"
	"ias/project/utils"
	"log"
	"net"
	"sync"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Server struct {
	client   *communication.WireguardClient
	conn     *net.UDPConn
	msgBuf   []byte
	sendLock sync.Mutex
}

func CreateServer(client *communication.WireguardClient, conn *net.UDPConn) *Server {
	return &Server{client: client, conn: conn, msgBuf: make([]byte, 1024)}
}

func (s *Server) Start() {
	go communication.ListenUDP(s.conn, s.handlePacket)
}

func (s *Server) Close() error {
	err := s.conn.Close()
	if err != nil {
		return err
	}
	return s.client.Close()
}

func (s *Server) handlePacket(message string, originAddr *net.UDPAddr) {
	query := utils.GetQuery(message)
	// add <public_key> <ip>
	if query[0] == "add" {
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		err = s.client.AddPeer(publicKey, query[2]+"/32", nil, false)
		if err != nil {
			log.Println(fmt.Errorf("AddPeer failed: %w", err))
			return
		}
		//Reply with add
		ownPublicKey, err := s.client.GetDevicePublicKey()
		if err != nil {
			log.Println(fmt.Errorf("GetDevicePublicKey failed: %w", err))
			return
		}
		interfaceIp, err := s.client.GetInterfaceIP()
		if err != nil {
			log.Println(fmt.Errorf("GetInterfaceIP failed: %w", err))
			return
		}
		s.sendLock.Lock()
		err = communication.SendUDPMessage(s.msgBuf, s.conn, fmt.Sprintf("add %s %s", *ownPublicKey, *interfaceIp), *originAddr, false)
		if err != nil {
			log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			s.sendLock.Unlock()
			return
		}
		s.sendLock.Unlock()
		log.Printf("Added peer %s and replied\n", query[1])
	}
	// get <public_key>
	if query[0] == "get" {
		publicKey, err := wgtypes.ParseKey(query[1])
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		peer, err := s.client.GetPeer(publicKey)
		if err != nil {
			log.Println(fmt.Errorf("GetPeer failed: %w", err))
			return
		}
		jsonData, err := json.Marshal(peer)
		if err != nil {
			log.Println(fmt.Errorf("encoding peer failed: %w", err))
			return
		}
		s.sendLock.Lock()
		err = communication.SendUDPMessage(
			s.msgBuf,
			s.conn,
			fmt.Sprintf("peer %s", string(jsonData)),
			*originAddr,
			false)
		if err != nil {
			log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			s.sendLock.Unlock()
			return
		}
		s.sendLock.Unlock()
		log.Printf("Returned peer data: %s\n", string(jsonData))
	}
}
