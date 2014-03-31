package netsec

import (
	"errors"
	"os/exec"
)

func executeCommand(command string, args ...string) (string, error) {
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

