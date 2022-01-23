package communication

import (
	"fmt"
	"net"
)

func CreateUDPSocket(port string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		return nil, err
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return sock, nil
}

func SendUDPMessage(conn *net.UDPConn, message string, address net.UDPAddr) error {
	n, err := conn.WriteTo([]byte(message), &address)
	if err != nil {
		return err
	}
	if n != 0 {
		return fmt.Errorf("connection WriteTo call returned non-zero error code (%d)", n)
	}
	return nil
}

func ListenUDP(conn *net.UDPConn, handleMessage func(content string, originAddr *net.UDPAddr)) error {
	buf := make([]byte, 1024)
	for {
		rlen, originAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		go handleMessage(string(buf[0:rlen]), originAddr)
	}
}

// func CreateConsumerRules(wgClient *WireguardClient) error {
// 	wgClient.SetFirewallMark(constants.DEFAULT_FIREWALL_MARK)
// 	firewallMarkString := strconv.Itoa(constants.DEFAULT_FIREWALL_MARK)
// 	err := utils.SudoExec("ip", "route", "add", "default", "dev", wgClient.iface, "table", firewallMarkString)
// 	if err != nil {
// 		return err
// 	}
// 	cmd := "echo 'nameserver 8.8.8.8' | sudo resolvconf -a tun." + wgClient.iface + " -m 0 -x"
// 	_, err = exec.Command("sudo", "bash", "-c", cmd).CombinedOutput()
// 	if err != nil {
// 		return err
// 	}
// 	return utils.SudoExec("ip", "rule", "add", "not", "fwmark", firewallMarkString, "table", firewallMarkString)
// }

// func RemoveConsumerRules(wgClient *WireguardClient) error {
// 	firewallMarkString := strconv.Itoa(constants.DEFAULT_FIREWALL_MARK)
// 	err := utils.SudoExec("ip", "route", "del", "default", "dev", wgClient.iface, "table", firewallMarkString)
// 	if err != nil {
// 		return err
// 	}
// 	err = utils.SudoExec("resolvconf", "-d", "tun."+wgClient.iface, "-f")
// 	if err != nil {
// 		return err
// 	}
// 	return utils.SudoExec("ip", "rule", "del", "not", "fwmark", firewallMarkString, "table", firewallMarkString)
// }
