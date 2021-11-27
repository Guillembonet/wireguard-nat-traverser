package communication

import (
	"fmt"
	"ias/project/utils"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WireguardClient struct {
	iface  string
	client *wgctrl.Client
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

func (wc *WireguardClient) SetInterfaceIP(hostId string) error {
	if err := utils.SudoExec("ip", "address", "replace", "dev", wc.iface, "10.0.0."+hostId); err != nil {
		return err
	}
	return nil
}

func (wc *WireguardClient) AddPeer(publicKey wgtypes.Key, cidr string, endpoint *net.UDPAddr) error {
	device, err := wc.client.Device(wc.iface)
	if err != nil {
		return err
	}
	_, peerIps, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	defaultKeepAlive := time.Hour * 5
	peer := wgtypes.PeerConfig{PublicKey: publicKey, PersistentKeepaliveInterval: &defaultKeepAlive, AllowedIPs: []net.IPNet{*peerIps}, Endpoint: endpoint}
	config := wgtypes.Config{
		PrivateKey: &device.PrivateKey,
		ListenPort: &device.ListenPort,
		Peers:      []wgtypes.PeerConfig{peer},
	}
	wc.client.ConfigureDevice(wc.iface, config)
	return nil
}

func peerConfig(peer wgtypes.Peer) wgtypes.PeerConfig {
	endpoint := peer.Endpoint
	publicKey := peer.PublicKey
	keepAliveInterval := peer.PersistentKeepaliveInterval
	allowedIPs := peer.AllowedIPs

	return wgtypes.PeerConfig{
		Endpoint:                    endpoint,
		PublicKey:                   publicKey,
		AllowedIPs:                  allowedIPs,
		PersistentKeepaliveInterval: &keepAliveInterval,
	}
}
