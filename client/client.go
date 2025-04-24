package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func New(config *Config) *Client {
	return &Client{
		Config:      config,
		Connected:   false,
		MessageChan: make(chan Message, 10),
	}
}

func (c *Client) Connect() error {
	serverAddr := fmt.Sprintf("[%s]:%s", c.Config.ServerHost, c.Config.ServerPort)
	if strings.Contains(c.Config.ServerHost, ":") && !strings.HasPrefix(c.Config.ServerHost, "[") {
		serverAddr = fmt.Sprintf("[%s]:%s", c.Config.ServerHost, c.Config.ServerPort)
	}
	log.Printf("Attempting to connect to server at %s", serverAddr)
	conn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", serverAddr, err)
	}
	log.Printf("Successfully connected to server %s", serverAddr)

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
	log.Printf("Sending registration request for user: %s", c.Config.Username)
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
	if !c.Connected || c.Conn == nil {
		return fmt.Errorf("not connected to server")
	}

	data, err := Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = c.Conn.Write(data)
	c.Conn.SetWriteDeadline(time.Time{})

	if err != nil {
		log.Printf("Failed to send request type %s: %v. Disconnecting.", req["type"], err)
		c.Disconnect()
		return fmt.Errorf("failed to send request: %w", err)
	}
	log.Printf("Sent request: Type=%s", req["type"])
	return nil
}

func (c *Client) handleIncomingMessages() {
	if c.Conn == nil {
		log.Println("Error: handleIncomingMessages called with nil connection.")
		return
	}
	defer func() {
		log.Println("handleIncomingMessages goroutine stopping.")
		if c.Conn != nil {
			c.Conn.Close()
		}
		c.Connected = false
	}()

	decoder := json.NewDecoder(c.Conn)
	for {
		var msg Message
		err := decoder.Decode(&msg)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Println("Read timeout on connection. Assuming disconnected.")
			} else {
				log.Printf("Connection closed or error decoding message: %v", err)
			}
			break
		}

		log.Printf("Received message: Type=%s, Payload=%+v", msg.Type, msg.Message)

		c.ProcessMessage(msg)

	}
}

func (c *Client) ProcessMessage(msg Message) {
	switch msg.Type {
	case "registration", "reconnect":
		log.Printf("Server response: %v", msg.Message["status"])
	case "player_list":
		c.processPlayerList(msg)
	case "match_start":
		c.processMatchStart(msg)
	case "game_start":
		c.processGameStart(msg)
	case "turn_request":
		c.handleTurnRequest(msg)
	case "switch_request":
		c.handleSwitchRequest(msg)
	case "turn_result":
		c.handleTurnResult(msg)
	case "opponent_disconnected":
		c.handleOpponentDisconnected(msg)
	case "game_end":
		c.processGameEnd(msg)
	case "match_error":
		errMsg, _ := msg.Message["error"].(string)
		fmt.Printf("\nMatchmaking Error: %s\n> ", errMsg)
	default:
		log.Printf("Unknown message type received: %s", msg.Type)
	}
}

func (c *Client) Disconnect() {
	if c.Conn != nil {
		log.Println("Disconnecting client...")
		c.Conn.Close()
		c.Conn = nil
	}
	c.Connected = false
	if c.GameActive || c.AwaitingForcedSwitch {
		c.endGameMode()
	}
}

func (c *Client) Run() {
	if err := c.Connect(); err != nil {
		fmt.Printf("Initial connection failed: %v\n", err)
		fmt.Println("You are disconnected. Use 'connect' to try again or 'quit' to exit.")
	} else {
		fmt.Println("Connected to server. Type 'help' for commands.")
	}

	prompt := "> "
	if !c.Connected {
		prompt = "(disconnected)> "
	}
	fmt.Print(prompt)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		if c.Connected {
			prompt = "> "
		} else {
			prompt = "(disconnected)> "
		}

		if input == "" {
			if !c.GameActive && !c.AwaitingForcedSwitch {
				fmt.Print(prompt)
			}
			continue
		}

		if c.GameActive || c.AwaitingForcedSwitch {
			if !c.Connected {
				fmt.Println("\nConnection lost during game action.")
				c.endGameMode()
				prompt = "(disconnected)> "
				fmt.Print(prompt)
				continue
			}
			c.handleGameInput(input)
			continue
		}

		args := strings.Fields(input)
		command := args[0]

		switch command {
		case "help":
			fmt.Println("\nAvailable commands:")
			if c.Connected {
				fmt.Println("  players          - List online players")
				fmt.Println("  match <username> - Challenge a player to a battle")
			} else {
				fmt.Println("  connect          - Attempt to connect/reconnect to the server")
			}
			fmt.Println("  quit             - Disconnect and exit")
			fmt.Print(prompt)

		case "quit":
			fmt.Println("Disconnecting and exiting...")
			c.Disconnect()
			fmt.Println("Exited.")
			return

		case "connect":
			if c.Connected {
				fmt.Println("Already connected.")
			} else {
				fmt.Println("Attempting to connect...")
				if err := c.Connect(); err != nil {
					fmt.Printf("Connection attempt failed: %v\n", err)
				} else {
					fmt.Println("Successfully connected and registered!")
				}
			}
			if c.Connected {
				fmt.Print("> ")
			} else {
				fmt.Print("(disconnected)> ")
			}

		case "players":
			if !c.Connected {
				fmt.Println("Not connected to server. Use 'connect' to try again.")
				fmt.Print(prompt)
				continue
			}
			if err := c.GetPlayers(); err != nil {
				fmt.Printf("Error getting players: %v\n", err)
			}

		case "match":
			if !c.Connected {
				fmt.Println("Not connected to server. Use 'connect' to try again.")
				fmt.Print(prompt)
				continue
			}
			if len(args) < 2 {
				fmt.Println("Usage: match <username>")
				fmt.Print(prompt)
				continue
			}
			opponent := args[1]
			if opponent == c.Config.Username {
				fmt.Println("You cannot match with yourself.")
				fmt.Print(prompt)
				continue
			}

			fmt.Printf("Attempting to match with %s...\n", opponent)
			if err := c.Matchmake(opponent); err != nil {
				fmt.Printf("Error starting match: %v\n", err)
			}

		default:
			fmt.Println("Unknown command. Type 'help' for a list of commands.")
			fmt.Print(prompt)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
	c.Disconnect()
	log.Println("Client Run loop finished.")
}
