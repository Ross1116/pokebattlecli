package main

import (
	"log"

	"github.com/ross1116/pokebattlecli/server"
)

func main() {
	config := server.Config{
		Host: "localhost",
		Port: "9090",
	}
	srv := server.New(&config)
	log.Printf("Starting server at %s:%s...", config.Host, config.Port)
	srv.Run()
}
