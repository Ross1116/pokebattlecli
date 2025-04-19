package main

import (
	"github.com/ross1116/pokebattlecli/server"
)

func main() {
	config := server.Config{
		Host: "localhost",
		Port: "9090",
	}
	srv := server.New(&config)
	srv.Run()
}
