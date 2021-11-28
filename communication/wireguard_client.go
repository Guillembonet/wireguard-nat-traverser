package communication

import (
	"fmt"
	"ias/project/utils"
	"log"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WireguardClient struct {
	iface  string
	client *wgctrl.Client
}

type PeerData struct {
	PublicKey string
	Ip        string
	Endpoint  string
}

func NewWireguardClient(iface string) (*WireguardClient, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	return &WireguardClient{
		iface:  iface,
		client: wgClient,
	}, nil
}

func (wc *WireguardClient) ConfigureWireguardClient(port int) error {
	key, _ := wgtypes.GenerateKey()

	err := wc.createDevice()
	if err != nil {
		return err
	}

	config := wgtypes.Config{
		PrivateKey: &key,
		ListenPort: &port,
		Peers:      []wgtypes.PeerConfig{},
	}

	err = wc.client.ConfigureDevice(wc.iface, config)
	if err != nil {
		return err
	}

	return nil
}

func (wc *WireguardClient) createDevice() error {
	if d, err := wc.client.Device(wc.iface); err != nil || d.Name != wc.iface {
		err := utils.SudoExec("ip", "link", "add", "dev", wc.iface, "type", "wireguard")
		if err != nil {
			return err
		}
	} else {
		//Rebuild
		err := utils.SudoExec("ip", "link", "del", "dev", wc.iface)
		if err != nil {
			return err
		}
		err = utils.SudoExec("ip", "link", "add", "dev", wc.iface, "type", "wireguard")
		if err != nil {
			return err
		}
	}
	err := utils.SudoExec("ip", "link", "set", "dev", wc.iface, "up")
	if err != nil {
		return err
	}
	err = utils.SudoExec("ip", "route", "add", "10.0.0.0/24", "dev", wc.iface)
	if err != nil {
		return err
	}

	return nil
}

func (wc *WireguardClient) destroyDevice() error {
	err := utils.SudoExec("ip", "route", "del", "10.0.0.0/24")
	if err != nil {
		return err
	}
	err = utils.SudoExec("ip", "link", "del", "dev", wc.iface)
	if err != nil {
		return err
	}
	return nil
}

func (wc *WireguardClient) Close() error {
	err := wc.destroyDevice()
	if err != nil {
		return err
	}
	err = wc.client.Close()
	if err != nil {
		return err
	}
	log.Println("Closed!")
	return nil
}

func (wc *WireguardClient) GetDevicePublicKey() (*string, error) {
	device, err := wc.client.Device(wc.iface)
	if err != nil {
		return nil, err
	}
	publicKey := device.PublicKey.String()
	return &publicKey, nil
}

func (wc *WireguardClient) GetInterfaceIP() (*string, error) {
	ief, err := net.InterfaceByName(wc.iface)
	if err != nil {
		return nil, err
	}
	addrs, err := ief.Addrs()
	if err != nil {
		return nil, err
	}
	if len(addrs) < 1 {
		return nil, fmt.Errorf("interface %s doesn't have an ip address", wc.iface)
	}
	ipv4Addr := addrs[0].(*net.IPNet).IP
	if ipv4Addr == nil {
		return nil, fmt.Errorf("interface %s has a null ip", wc.iface)
	}
	ip := ipv4Addr.String()
	return &ip, nil
}

func (wc *WireguardClient) SetInterfaceIP(ip string) error {
	if err := utils.SudoExec("ip", "address", "flush", "dev", wc.iface); err != nil {
		return err
	}
	if err := utils.SudoExec("ip", "address", "replace", "dev", wc.iface, ip); err != nil {
		return err
	}
	return nil
}

func (wc *WireguardClient) AddPeer(publicKey wgtypes.Key, cidr string, endpoint *net.UDPAddr, replacePeers bool) error {
	device, err := wc.client.Device(wc.iface)
	if err != nil {
		return err
	}
	_, peerIps, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	defaultKeepAlive := time.Second * 5
	peer := wgtypes.PeerConfig{PublicKey: publicKey, PersistentKeepaliveInterval: &defaultKeepAlive, AllowedIPs: []net.IPNet{*peerIps}, Endpoint: endpoint}
	config := wgtypes.Config{
		PrivateKey:   &device.PrivateKey,
		ListenPort:   &device.ListenPort,
		Peers:        []wgtypes.PeerConfig{peer},
		ReplacePeers: replacePeers,
	}
	err = wc.client.ConfigureDevice(wc.iface, config)
	if err != nil {
		return err
	}
	return nil
}

func (wc *WireguardClient) RemovePeerByAllowedIP(allowedIP string) error {
	device, err := wc.client.Device(wc.iface)
	if err != nil {
		return err
	}
	peerConfigs := []wgtypes.PeerConfig{}
	for _, p := range device.Peers {
		hasAllowedIP := false
		for _, a := range p.AllowedIPs {
			if a.String() == allowedIP {
				hasAllowedIP = true
			}
		}
		if hasAllowedIP {
			continue
		}
		peerConfigs = append(peerConfigs, wgtypes.PeerConfig{
			PublicKey:                   p.PublicKey,
			AllowedIPs:                  p.AllowedIPs,
			Endpoint:                    p.Endpoint,
			PersistentKeepaliveInterval: &p.PersistentKeepaliveInterval,
		})
	}
	config := wgtypes.Config{
		PrivateKey:   &device.PrivateKey,
		ListenPort:   &device.ListenPort,
		Peers:        peerConfigs,
		ReplacePeers: true,
	}
	err = wc.client.ConfigureDevice(wc.iface, config)
	if err != nil {
		return err
	}
	return nil
}

func (wc *WireguardClient) GetPeerIPAndEndpoint(publicKey wgtypes.Key) (string, string, error) {
	device, err := wc.client.Device(wc.iface)
	if err != nil {
		return "", "", err
	}
	for _, p := range device.Peers {
		if p.PublicKey == publicKey && p.Endpoint != nil && p.AllowedIPs != nil && len(p.AllowedIPs) > 0 {
			return p.AllowedIPs[0].String(), p.Endpoint.String(), nil
		}
	}
	return "", "", fmt.Errorf("peer %s not found", publicKey.String())
}
