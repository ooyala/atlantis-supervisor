/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

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
