package communication

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func HandleAdd(publicKeyString string, hostId string, addPear func(wgtypes.Key, string, *net.UDPAddr) error, endpoint *net.UDPAddr) error {
	publicKey, err := wgtypes.ParseKey(publicKeyString)
	if err != nil {
		return err
	}
	ip := hostId
	err = addPear(publicKey, ip, endpoint)
	if err != nil {
		return err
	}
	fmt.Printf("Added peer %s with ip %s\n", publicKey, ip)
	return nil
}
