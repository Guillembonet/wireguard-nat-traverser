package communication

// import (
// 	"fmt"
// 	"ias/project/constants"
// 	"ias/project/utils"
// 	"net"

// 	"golang.zx2c4.com/wireguard/wgctrl"
// 	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
// )

// type WireguardClient struct {
// 	iface  string
// 	client *wgctrl.Client
// }

// type PeerData struct {
// 	PublicKey string
// 	CIDR      string
// 	Endpoint  string
// }

// func NewWireguardClient(iface string) (*WireguardClient, error) {
// 	wgClient, err := wgctrl.New()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &WireguardClient{
// 		iface:  iface,
// 		client: wgClient,
// 	}, nil
// }

// func (wc *WireguardClient) ConfigureWireguardClient(port int) error {
// 	key, err := wgtypes.GenerateKey()
// 	if err != nil {
// 		return err
// 	}
// 	err = wc.createDevice()
// 	if err != nil {
// 		return err
// 	}
// 	config := wgtypes.Config{
// 		PrivateKey: &key,
// 		ListenPort: &port,
// 		Peers:      []wgtypes.PeerConfig{},
// 	}

// 	return wc.client.ConfigureDevice(wc.iface, config)
// }

// func (wc *WireguardClient) createDevice() error {
// 	if d, err := wc.client.Device(wc.iface); err != nil || d.Name != wc.iface {
// 		err := utils.SudoExec("ip", "link", "add", "dev", wc.iface, "type", "wireguard")
// 		if err != nil {
// 			return err
// 		}
// 	} else {
// 		// Recreate
// 		err := utils.SudoExec("ip", "link", "del", "dev", wc.iface)
// 		if err != nil {
// 			return err
// 		}
// 		err = utils.SudoExec("ip", "link", "add", "dev", wc.iface, "type", "wireguard")
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	err := utils.SudoExec("ip", "link", "set", "dev", wc.iface, "up")
// 	if err != nil {
// 		return err
// 	}
// 	return utils.SudoExec("ip", "route", "add", constants.DEFAULT_BASE_IP+"0/24", "dev", wc.iface)
// }

// func (wc *WireguardClient) destroyDevice() error {
// 	err := utils.SudoExec("ip", "route", "del", constants.DEFAULT_BASE_IP+"0/24")
// 	if err != nil {
// 		return err
// 	}
// 	return utils.SudoExec("ip", "link", "del", "dev", wc.iface)
// }

// func (wc *WireguardClient) Close() error {
// 	err := wc.destroyDevice()
// 	if err != nil {
// 		return err
// 	}
// 	return wc.client.Close()
// }

// func (wc *WireguardClient) GetDevicePublicKey() (*string, error) {
// 	device, err := wc.client.Device(wc.iface)
// 	if err != nil {
// 		return nil, err
// 	}
// 	publicKey := device.PublicKey.String()
// 	return &publicKey, nil
// }

// func (wc *WireguardClient) GetInterfaceIP() (*string, error) {
// 	ief, err := net.InterfaceByName(wc.iface)
// 	if err != nil {
// 		return nil, err
// 	}
// 	addrs, err := ief.Addrs()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(addrs) != 1 {
// 		return nil, fmt.Errorf("interface %s doesn't have an ip address or has invalid config", wc.iface)
// 	}
// 	ipv4Addr := addrs[0].(*net.IPNet).IP
// 	if ipv4Addr == nil {
// 		return nil, fmt.Errorf("interface %s has a null ip", wc.iface)
// 	}
// 	ip := ipv4Addr.String()
// 	return &ip, nil
// }

// func (wc *WireguardClient) SetInterfaceIP(ip string) error {
// 	if err := utils.SudoExec("ip", "address", "flush", "dev", wc.iface); err != nil {
// 		return err
// 	}
// 	return utils.SudoExec("ip", "address", "replace", "dev", wc.iface, ip)
// }

// func (wc *WireguardClient) AddPeer(publicKey wgtypes.Key, cidr string, endpoint *net.UDPAddr, replacePeers bool) error {
// 	device, err := wc.client.Device(wc.iface)
// 	if err != nil {
// 		return err
// 	}
// 	_, peerIps, err := net.ParseCIDR(cidr)
// 	if err != nil {
// 		return err
// 	}
// 	keepalive := constants.DEFAULT_KEEPALIVE
// 	peer := wgtypes.PeerConfig{PublicKey: publicKey, PersistentKeepaliveInterval: &keepalive, AllowedIPs: []net.IPNet{*peerIps}, Endpoint: endpoint}
// 	config := wgtypes.Config{
// 		PrivateKey:   &device.PrivateKey,
// 		ListenPort:   &device.ListenPort,
// 		Peers:        []wgtypes.PeerConfig{peer},
// 		ReplacePeers: replacePeers,
// 	}
// 	return wc.client.ConfigureDevice(wc.iface, config)
// }

// func (wc *WireguardClient) RemovePeerByAllowedIP(allowedIP string) error {
// 	device, err := wc.client.Device(wc.iface)
// 	if err != nil {
// 		return err
// 	}
// 	peerConfigs := []wgtypes.PeerConfig{}
// 	for _, p := range device.Peers {
// 		hasAllowedIP := false
// 		for _, a := range p.AllowedIPs {
// 			if a.String() == allowedIP {
// 				hasAllowedIP = true
// 			}
// 		}
// 		if !hasAllowedIP {
// 			peerConfigs = append(peerConfigs, wgtypes.PeerConfig{
// 				PublicKey:                   p.PublicKey,
// 				AllowedIPs:                  p.AllowedIPs,
// 				Endpoint:                    p.Endpoint,
// 				PersistentKeepaliveInterval: &p.PersistentKeepaliveInterval,
// 			})
// 		}
// 	}
// 	config := wgtypes.Config{
// 		Peers:        peerConfigs,
// 		ReplacePeers: true,
// 	}
// 	return wc.client.ConfigureDevice(wc.iface, config)
// }

// func (wc *WireguardClient) GetPeer(publicKey wgtypes.Key) (*PeerData, error) {
// 	device, err := wc.client.Device(wc.iface)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, p := range device.Peers {
// 		if p.PublicKey == publicKey && p.Endpoint != nil && p.AllowedIPs != nil && len(p.AllowedIPs) > 0 {
// 			return &PeerData{PublicKey: publicKey.String(), CIDR: p.AllowedIPs[0].String(), Endpoint: p.Endpoint.String()}, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("peer %s not found", publicKey.String())
// }

// func (wc *WireguardClient) SetFirewallMark(mark int) error {
// 	config := wgtypes.Config{
// 		FirewallMark: &mark,
// 	}
// 	return wc.client.ConfigureDevice(wc.iface, config)
// }
