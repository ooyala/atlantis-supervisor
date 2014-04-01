package netsec

import (
	"errors"
	"log"
	"os/exec"
)

func executeCommand(command string, args ...string) (string, error) {
	cmdStr := command
	for _, arg := range args {
		cmdStr += " " + arg
	}
	log.Println("[netsec] " + cmdStr)
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			return string(out), errors.New(string(out))
		default:
			return string(out), err
		}
	}
	return string(out), nil
}
