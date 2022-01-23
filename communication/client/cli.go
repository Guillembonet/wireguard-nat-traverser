package client

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/guillembonet/wireguard-nat-traverser/communication"
	"github.com/guillembonet/wireguard-nat-traverser/constants"
	"github.com/guillembonet/wireguard-nat-traverser/utils"
	"github.com/urfave/cli"
)

func (c *Client) cli() {
	app := &cli.App{
		Name: "client",
		Commands: []cli.Command{
			{
				Name:        constants.AddQuery,
				Aliases:     []string{"a"},
				UsageText:   "add [host-id]",
				Usage:       "add a connection to the server",
				Description: "[host-id] must be a number between 1 and 256",
				Action: func(ctx *cli.Context) error {
					hostId := ctx.Args().Get(1)
					hostIdInt, err := strconv.Atoi(hostId)
					if ctx.Args().First() == constants.HelpQuery || hostId == "" || err != nil || hostIdInt < 1 || hostIdInt > 256 {
						cli.ShowCommandHelp(ctx, ctx.Command.Name)
						return nil
					}
					publicKey, err := c.manager.GetPublicKey()
					if err != nil {
						return fmt.Errorf("getting public key failed: %w", err)
					}
					ipPreffix := strings.Join(strings.Split(c.params.TunnelSlash24IP, ".")[0:2], ".")
					err = c.manager.SetInterfaceIP(ipPreffix + hostId + "/24")
					if err != nil {
						return fmt.Errorf("SetInterfaceIP failed: %w", err)
					}
					interfaceIp, err := c.manager.GetInterfaceIP()
					if err != nil {
						return fmt.Errorf("GetInterfaceIP failed: %w", err)
					}
					err = communication.SendUDPMessage(c.conn, fmt.Sprintf("add %s %s", *publicKey, interfaceIp), *c.server)
					if err != nil {
						return fmt.Errorf("SendUDPMessage failed: %w", err)
					}
					return nil
				},
			},
		},
	}
	reader := bufio.NewReader(os.Stdin)

	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Println(fmt.Errorf("read command error: %w", err))
			break
		}
		query := utils.GetQuery(text)
		err = app.Run(append([]string{"client"}, query...))
		if err != nil {
			log.Println(fmt.Errorf("command error: %w", err))
			break
		}
	}
}

// case "add":
// 	publicKey, err := c.client.GetDevicePublicKey()
// 	if err != nil {
// 		log.Println(fmt.Errorf("GetDevicePublicKey failed: %w", err))
// 		break
// 	}
// 	hostId := query[1]
// 	err = c.client.SetInterfaceIP(constants.DEFAULT_BASE_IP + hostId + "/24")
// 	if err != nil {
// 		log.Println(fmt.Errorf("SetInterfaceIP failed: %w", err))
// 		break
// 	}
// 	interfaceIp, err := c.client.GetInterfaceIP()
// 	if err != nil {
// 		log.Println(fmt.Errorf("GetInterfaceIP failed: %w", err))
// 		break
// 	}
// 	err = communication.SendUDPMessage(msgBuf, c.conn, fmt.Sprintf("add %s %s", *publicKey, *interfaceIp), *c.serverAddr, false)
// 	if err != nil {
// 		log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
// 	}
// //connect <public_key>
// case "connect":
// 	err := communication.SendUDPMessage(msgBuf, c.conn, "get "+query[1], *c.serverAddr, false)
// 	if err != nil {
// 		log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
// 	}
// // remove
// case "remove":
// 	c.client.RemovePeerByAllowedIP(c.serverAddr.String() + "/32")
// 	c.serverAddr = c.initialServerAddr
// 	log.Println("Removed server connection")
// // set consumer|provider
// case "set":
// 	if strings.HasPrefix(query[1], "c") {
// 		c.isConsumer = true
// 		log.Println("Consumer mode set")
// 		break
// 	}
// 	if strings.HasPrefix(query[1], "p") {
// 		c.isConsumer = false
// 		log.Println("Provider mode set")
// 	}
// // exit
// case "exit":
// 	err := communication.SendUDPMessage(msgBuf, c.conn, "exit", *c.serverAddr, false)
// 	if err != nil {
// 		log.Println(fmt.Errorf("SendUDPMessage failed: %w", err))
// 	}
// 	c.finished <- true
// }
