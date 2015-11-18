package helper

import (
        "testing"
	"fmt"
)

func TestHostLogDir(t *testing.T) {
   out := HostLogDir("test")
   if out != "/var/log/atlantis/containers/test" {
      fmt.Printf("helper::HostLogDir expecting /var/log/atlantis/containers/test but got %s", out)
      t.Fail()
   }
}

func TestHostConfigDir(t *testing.T) {
   out := HostConfigDir("test")
   if out != "/etc/atlantis/containers/test" {
      fmt.Printf("helper::HostLogDir expecting /etc/atlantis/containers/test but got %s", out)
      t.Fail()
   }
}

func TestHostConfigFile(t *testing.T) {
   out := HostConfigFile("test")
   if out != "/etc/atlantis/containers/test/config.json" {
      fmt.Printf("helper::HostLogDir expecting /test/config.json but got %s", out)
      t.Fail()
   }
}
