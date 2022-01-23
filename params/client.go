package params

import "flag"

type Client struct {
	ServerIP            string
	ServerUDPPort       int
	ServerWireguardPort int
	TunnelSlash24IP     string
}

func (c *Client) Init() {
	flag.String("serverIp", "empty", "IP address of the server")
	flag.Int("serverUdpPort", 2001, "port used by the server for udp communication")
	flag.Int("serverWireguardPort", 2001, "port used by the server for the wireguard interface")
	flag.String("tunnelSlash24IP", "10.1.0.0", "cidr of the tunnel network (example: 10.0.1.0)")
}
