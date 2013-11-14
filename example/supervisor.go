package main

import (
	"atlantis/supervisor/server"
)

func main() {
	srv := server.New()
	srv.Run()
}
