package params

import "flag"

type Generic struct {
	UDPPort       int
	WireguardPort int
	InterfaceName string
}

func (g *Generic) Init() {
	flag.Int("udpPort", 2000, "udp port used for communication")
	flag.Int("wireguardPort", 2001, "port used for the wireguard interface")
	flag.String("interfaceName", "wg0", "name of the wireguard interface")
}
