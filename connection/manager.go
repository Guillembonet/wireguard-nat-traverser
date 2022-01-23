package connection

import (
	"fmt"
	"net"

	"github.com/guillembonet/wireguard-nat-traverser/constants"
	"github.com/guillembonet/wireguard-nat-traverser/utils"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Manager is in charge of managing the wireguard connections
type Manager struct {
	iface     string
	client    *wgctrl.Client
	publicKey *wgtypes.Key
}

func NewManager(iface string) (*Manager, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	return &Manager{
		iface:  iface,
		client: wgClient,
	}, nil
}

// Initializes the wireguard client with a new key and creates a device for it
func (m *Manager) Initialize(port int) error {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return err
	}
	m.publicKey = &key
	err = m.createDevice()
	if err != nil {
		return err
	}
	config := wgtypes.Config{
		PrivateKey: &key,
		ListenPort: &port,
		Peers:      []wgtypes.PeerConfig{},
	}

	return m.client.ConfigureDevice(m.iface, config)
}

func (m *Manager) createDevice() error {
	if d, err := m.client.Device(m.iface); err != nil || d.Name != m.iface {
		err := utils.SudoExec("ip", "link", "add", "dev", m.iface, "type", "wireguard")
		if err != nil {
			return err
		}
		err = utils.SudoExec("ip", "link", "set", "dev", m.iface, "up")
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("device with (interface name = %s) already exists", m.iface)
}

// AddPeer adds a new wireguard peer
func (m *Manager) AddPeer(publicKey wgtypes.Key, cidr string, endpoint *net.UDPAddr, replacePeers bool) error {
	device, err := m.client.Device(m.iface)
	if err != nil {
		return err
	}
	_, peerIps, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	keepalive := constants.DEFAULT_KEEPALIVE
	peer := wgtypes.PeerConfig{PublicKey: publicKey, PersistentKeepaliveInterval: &keepalive, AllowedIPs: []net.IPNet{*peerIps}, Endpoint: endpoint}
	config := wgtypes.Config{
		PrivateKey:   &device.PrivateKey,
		ListenPort:   &device.ListenPort,
		Peers:        []wgtypes.PeerConfig{peer},
		ReplacePeers: replacePeers,
	}
	return m.client.ConfigureDevice(m.iface, config)
}

func (m *Manager) destroyDevice() error {
	return utils.SudoExec("ip", "link", "del", "dev", m.iface)
}

func (m *Manager) Cleanup() error {
	err := m.destroyDevice()
	if err != nil {
		return err
	}
	return m.client.Close()
}

func (m *Manager) GetInterfaceIP() (string, error) {
	iface, err := net.InterfaceByName(m.iface)
	if err != nil {
		return "", err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	if len(addrs) != 1 {
		return "", fmt.Errorf("interface %s doesn't have an ip address or has invalid config", m.iface)
	}
	ipv4Addr := addrs[0].(*net.IPNet).IP
	if ipv4Addr == nil {
		return "", fmt.Errorf("interface %s has a null ip", m.iface)
	}
	return ipv4Addr.String(), nil
}

func (m *Manager) GetPublicKey() (*wgtypes.Key, error) {
	device, err := m.client.Device(m.iface)
	if err != nil {
		return nil, err
	}
	return &device.PublicKey, nil
}

func (m *Manager) GetPeer(publicKey wgtypes.Key) (Peer, error) {
	device, err := m.client.Device(m.iface)
	if err != nil {
		return Peer{}, err
	}
	for _, p := range device.Peers {
		if p.PublicKey == publicKey && p.Endpoint != nil && p.AllowedIPs != nil && len(p.AllowedIPs) > 0 {
			return Peer{PublicKey: publicKey.String(), CIDR: p.AllowedIPs[0].String(), Endpoint: p.Endpoint.String()}, nil
		}
	}
	return Peer{}, fmt.Errorf("peer %s not found", publicKey.String())
}

func (m *Manager) SetInterfaceIP(ip string) error {
	if err := utils.SudoExec("ip", "address", "flush", "dev", m.iface); err != nil {
		return err
	}
	return utils.SudoExec("ip", "address", "replace", "dev", m.iface, ip)
}
