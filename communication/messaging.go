package communication

import (
	"fmt"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func HandleAdd(publicKeyString string, hostId string, addPear func(wgtypes.Key, string) error) error {
	publicKey, err := wgtypes.ParseKey(publicKeyString)
	if err != nil {
		return err
	}
	ip := hostId
	addPear(publicKey, ip)
	fmt.Printf("Added peer %s with ip %s\n", publicKey, ip)
	return nil
}
