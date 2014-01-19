package helper

import (
	"fmt"
)

func HostLogDir(cid string) string {
	return fmt.Sprintf("/var/log/atlantis/containers/%s", cid)
}

func HostConfigDir(cid string) string {
	return fmt.Sprintf("/etc/atlantis/containers/%s", cid)
}

func HostConfigFile(cid string) string {
	return fmt.Sprintf("%s/config.json", HostConfigDir(cid))
}
