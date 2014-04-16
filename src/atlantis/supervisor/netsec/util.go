package netsec

import (
	"errors"
	"log"
	"os/exec"
	"strings"
)

func echoIPTables(pretend bool) {
	executeCommand(pretend, "iptables", "-L")
	executeCommand(pretend, "iptables", "-t", "mangle", "-L")
}

func executeCommand(pretend bool, command string, args ...string) (string, error) {
	cmdStr := command
	for _, arg := range args {
		cmdStr += " " + arg
	}
	log.Println("[netsec][exec] " + cmdStr)
	var out string
	var err error
	if pretend {
		out = "pretending. no output."
	} else {
		cmd := exec.Command(command, args...)
		var outBytes []byte
		outBytes, err = cmd.CombinedOutput()
		out = string(outBytes)
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		log.Println("[netsec][exec] " + line)
	}
	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			return out, errors.New(out)
		default:
			return out, err
		}
	}
	return out, nil
}
