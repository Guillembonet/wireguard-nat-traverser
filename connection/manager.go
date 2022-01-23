package connection

import (
	"fmt"
	"ias/project/utils"

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
