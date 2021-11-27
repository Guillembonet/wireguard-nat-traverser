package utils

import (
	"os/exec"
)

func SudoExec(args ...string) error {
	_, err := exec.Command("sudo", args...).CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
