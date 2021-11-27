package communication

import (
	"fmt"
	"net"
)

func CreateUDPSocket(port string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		return nil, err
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return sock, nil
}

func SendUDPMessage(msgBuf []byte, conn *net.UDPConn, message string, address net.UDPAddr, printRes bool) error {
	copy(msgBuf, []byte(message))
	_, err := conn.WriteTo(msgBuf[:len(message)], &address)
	if err != nil {
		return err
	}

	if printRes {
		fmt.Printf("Message for %s\nContent: %s\n",
			address.String(), msgBuf[:len(message)])
	}
	return nil
}
