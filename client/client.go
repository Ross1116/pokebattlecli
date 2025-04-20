package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func New(config *Config) *Client {
	return &Client{
		Config:           config,
		Connected:        false,
		MessageChan:      make(chan Message, 10),
		GameInputChannel: make(chan string, 5),
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", fmt.Sprintf("[%s]:%s", c.Config.ServerHost, c.Config.ServerPort))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.Conn = conn
	c.Connected = true

	go c.handleIncomingMessages()

	return c.Register()
}

func (c *Client) Register() error {
	req := map[string]string{
		"type":     "register",
		"username": c.Config.Username,
	}
	return c.SendRequest(req)
}

func (c *Client) GetPlayers() error {
	req := map[string]string{
		"type":     "get_players",
		"username": c.Config.Username,
	}
	return c.SendRequest(req)
}

func (c *Client) Matchmake(opponent string) error {
	req := map[string]string{
		"type":     "matchmake",
		"username": c.Config.Username,
		"opponent": opponent,
	}
	return c.SendRequest(req)
}

func (c *Client) SendRequest(req map[string]string) error {
	if !c.Connected {
		return fmt.Errorf("not connected to server")
	}

	data, err := Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = c.Conn.Write(data)
	if err != nil {
		c.Connected = false
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}

func (c *Client) handleIncomingMessages() {
	defer c.Conn.Close()
	defer func() { c.Connected = false }()

	buffer := make([]byte, 1024)

	for {
		n, err := c.Conn.Read(buffer)
		if err != nil {
			log.Printf("Connection closed: %v", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		c.ProcessMessage(msg)

		c.MessageChan <- msg
	}
}

func (c *Client) ProcessMessage(msg Message) {
	switch msg.Type {
	case "registration":
		log.Printf("Registration successful: %v", msg.Message["status"])
	case "reconnect":
		log.Printf("Reconnected: %v", msg.Message["status"])
	case "player_list":
		c.processPlayerList(msg)
	case "match_start":
		c.processMatchStart(msg)
	case "game_start":
		c.processGameStart(msg)
	case "game_end":
		c.processGameEnd(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

func (c *Client) Disconnect() {
	if c.Connected && c.Conn != nil {
		c.Conn.Close()
		c.Connected = false
	}
}

func (c *Client) Run() {
	fmt.Println("Connected to server. Type 'help' for commands.")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		args := strings.Fields(input)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "help":
			fmt.Println("Available commands:")
			fmt.Println("  players - List all connected players")
			fmt.Println("  match <username> - Start a match with the specified player")
			fmt.Println("  quit - Disconnect and exit")
		case "players":
			if err := c.GetPlayers(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "match":
			if len(args) < 2 {
				fmt.Println("Usage: match <username>")
				continue
			}
			if err := c.Matchmake(args[1]); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "quit":
			c.Disconnect()
			return
		default:
			fmt.Println("Unknown command. Type 'help' for a list of commands.")
		}
	}
}
