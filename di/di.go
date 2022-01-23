package di

import (
	"strconv"
	"sync"

	"github.com/guillembonet/wireguard-nat-traverser/communication"
	"github.com/guillembonet/wireguard-nat-traverser/communication/client"
	"github.com/guillembonet/wireguard-nat-traverser/communication/server"
	"github.com/guillembonet/wireguard-nat-traverser/connection"
	"github.com/guillembonet/wireguard-nat-traverser/params"
)

// Container represents our dependency container
type Container struct {
	cleanup []func()
	lock    sync.Mutex
}

// Cleanup performs the cleanup required
func (c *Container) Cleanup() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for i := len(c.cleanup) - 1; i >= 0; i-- {
		c.cleanup[i]()
	}
}

// ConstructServer creates a server for us
func (c *Container) ConstructServer(gparams params.Generic) (*server.Server, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	manager, err := connection.NewManager(gparams.InterfaceName)
	if err != nil {
		return nil, err
	}
	err = manager.Initialize(gparams.WireguardPort)
	if err != nil {
		return nil, err
	}
	c.cleanup = append(c.cleanup, func() { manager.Cleanup() })
	sock, err := communication.CreateUDPSocket(":" + strconv.Itoa(gparams.UDPPort))
	if err != nil {
		return nil, err
	}
	c.cleanup = append(c.cleanup, func() { sock.Close() })
	server := server.NewServer(manager, sock)
	return server, nil
}

// ConstructServer creates a server for us
func (c *Container) ConstructClient(gparams params.Generic, cparams params.Client) (*client.Client, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	manager, err := connection.NewManager(gparams.InterfaceName)
	if err != nil {
		return nil, err
	}
	err = manager.Initialize(gparams.WireguardPort)
	if err != nil {
		return nil, err
	}
	c.cleanup = append(c.cleanup, func() { manager.Cleanup() })
	sock, err := communication.CreateUDPSocket(":" + strconv.Itoa(gparams.UDPPort))
	if err != nil {
		return nil, err
	}
	c.cleanup = append(c.cleanup, func() { sock.Close() })
	client := client.NewClient(manager, sock, cparams)
	return client, nil
}
