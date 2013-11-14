package main

import (
	"atlantis/supervisor/client"
)

func main() {
	cli := client.New()
	cli.Run()
}
