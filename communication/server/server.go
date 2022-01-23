package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/guillembonet/wireguard-nat-traverser/communication"
	"github.com/guillembonet/wireguard-nat-traverser/connection"
	"github.com/guillembonet/wireguard-nat-traverser/constants"
	"github.com/guillembonet/wireguard-nat-traverser/utils"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Server struct {
	manager connectionManager
	conn    *net.UDPConn
	lock    sync.Mutex
}

type connectionManager interface {
	AddPeer(publicKey wgtypes.Key, cidr string, endpoint *net.UDPAddr, replacePeers bool) error
	GetPublicKey() (*wgtypes.Key, error)
	GetInterfaceIP() (string, error)
	Cleanup() error
	GetPeer(wgtypes.Key) (connection.Peer, error)
}

func NewServer(manager connectionManager, conn *net.UDPConn) *Server {
	return &Server{manager: manager, conn: conn}
}

func (s *Server) Start() error {
	return communication.ListenUDP(s.conn, s.handlePacket)
}

func (s *Server) handlePacket(message string, originAddr *net.UDPAddr) {
	args := utils.GetQuery(message)
	switch args[0] {
	// add <public_key> <ip>
	case constants.AddQuery:
		publicKey, err := wgtypes.ParseKey(args[1])
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		err = s.manager.AddPeer(publicKey, args[2]+"/32", nil, false)
		if err != nil {
			log.Println(fmt.Errorf("AddPeer failed: %w", err))
			return
		}
		//Reply with add
		ownPublicKey, err := s.manager.GetPublicKey()
		if err != nil {
			log.Println(fmt.Errorf("GetDevicePublicKey failed: %w", err))
			return
		}
		interfaceIp, err := s.manager.GetInterfaceIP()
		if err != nil {
			log.Println(fmt.Errorf("GetInterfaceIP failed: %w", err))
			return
		}
		s.lock.Lock()
		defer s.lock.Unlock()
		err = communication.SendUDPMessage(s.conn, fmt.Sprintf("add %s %s", ownPublicKey.String(), interfaceIp), *originAddr)
		if err != nil {
			log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			return
		}
		log.Printf("Added peer %s and replied\n", args[1])
		return
	// get <public_key>
	case constants.GetQuery:
		publicKey, err := wgtypes.ParseKey(args[1])
		if err != nil {
			log.Println(fmt.Errorf("public key parsing failed: %w", err))
			return
		}
		peer, err := s.manager.GetPeer(publicKey)
		if err != nil {
			log.Println(fmt.Errorf("GetPeer failed: %w", err))
			return
		}
		jsonData, err := json.Marshal(peer)
		if err != nil {
			log.Println(fmt.Errorf("encoding peer failed: %w", err))
			return
		}
		s.lock.Lock()
		defer s.lock.Unlock()
		err = communication.SendUDPMessage(s.conn, fmt.Sprintf("peer %s", string(jsonData)), *originAddr)
		if err != nil {
			log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
			return
		}
		log.Printf("Returned peer data: %s\n", string(jsonData))
		return
	}
}
