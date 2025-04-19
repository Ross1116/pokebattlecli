package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ross1116/pokebattlecli/client"
)

func main() {
	serverHost := flag.String("host", "localhost", "Server host address")
	serverPort := flag.String("port", "9090", "Server port")
	username := flag.String("user", "", "Your username")

	flag.Parse()

	if *username == "" {
		fmt.Println("Please provide a username with -user flag")
		flag.Usage()
		os.Exit(1)
	}

	config := &client.Config{
		ServerHost: *serverHost,
		ServerPort: *serverPort,
		Username:   *username,
	}

	c := client.New(config)

	setupSignalHandler(c)

	fmt.Printf("Connecting to server %s:%s as %s...\n", *serverHost, *serverPort, *username)
	if err := c.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("Connected successfully!")

	c.Run()
}

func setupSignalHandler(c *client.Client) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nDisconnecting...")
		c.Disconnect()
		os.Exit(0)
	}()
}
