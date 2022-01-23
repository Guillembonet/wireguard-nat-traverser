package params

import "flag"

type Client struct {
	ServerIP            string
	ServerUDPPort       int
	ServerWireguardPort int
}

func (c *Client) Init() {
	flag.String("serverIp", "empty", "IP address of the server")
	flag.Int("serverUdpPort", 2001, "port used by the server for udp communication")
	flag.Int("serverWireguardPort", 2001, "port used by the server for the wireguard interface")
}
