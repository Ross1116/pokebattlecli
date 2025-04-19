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
}

type Client struct {
	conn     net.Conn
	username string
}

type Config struct {
	Host string
	Port string
}

func New(config *Config) *Server {
	return &Server{
		host:    config.Host,
		port:    config.Port,
		clients: make(map[string]*Client),
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

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go server.handleClient(conn)
	}
}

func (server *Server) SendResponse(conn net.Conn, response map[string]string) {
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
