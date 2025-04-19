package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type Server struct {
	host    string
	port    string
	clients map[string]*Client
	lobbies map[string]*Lobby
}

type Client struct {
	conn     net.Conn
	username string
}

type Lobby struct {
	player1 *Client
	player2 *Client
}

type Config struct {
	Host string
	Port string
}

type Response struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}

func New(config *Config) *Server {
	return &Server{
		host:    config.Host,
		port:    config.Port,
		clients: make(map[string]*Client),
		lobbies: make(map[string]*Lobby),
	}
}

func (s *Server) Clients() map[string]*Client {
	return s.clients
}

func (c *Client) Username() string {
	return c.username
}

func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	log.Println("Server started on", server.host, ":", server.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go server.handleClient(conn)
	}
}

func (server *Server) SendResponse(conn net.Conn, response Response) {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Println("Error marshaling response:", err)
		return
	}

	_, err = conn.Write(responseBytes)
	if err != nil {
		log.Println("Failed to send response:", err)
	}
}

func (server *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	// Read incoming request from the client
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading from client:", err)
		return
	}

	var msg map[string]string
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		log.Println("Invalid message format from client:", err)
		return
	}

	// Handle the request type
	switch msg["type"] {
	case "register":
		server.handleRegistration(msg, conn)
	case "get_players":
		server.handleGetPlayers(conn)
	case "matchmake":
		server.handleMatchmake(msg, conn)
	default:
		log.Println("Unknown message type:", msg["type"])
	}
}
