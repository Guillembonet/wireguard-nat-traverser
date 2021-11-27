package main

import "ias/project/utils"

func main() {
	utils.SudoExec("ip", "link", "del", "dev", "wg0")
	utils.SudoExec("ip", "link", "del", "dev", "wg1")
	utils.SudoExec("ip", "route", "del", "10.0.0.0/24")
}
