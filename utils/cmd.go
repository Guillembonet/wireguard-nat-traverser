package utils

import (
	"log"
	"os/exec"
)

func SudoExec(args ...string) error {
	out, err := exec.Command("sudo", args...).CombinedOutput()
	if err != nil {
		return err
	}
	log.Print(string(out))
	return nil
}
