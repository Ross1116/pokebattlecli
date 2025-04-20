package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
)

func New(config *Config) *Server {
	return &Server{
		host:    config.Host,
		port:    config.Port,
		clients: make(map[string]*Client),
		Lobbies: make(map[string]*Lobby),
	}
}

func (s *Server) Clients() map[string]*Client {
	return s.clients
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
		go server.HandleClient(conn)
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

func (server *Server) HandleClient(conn net.Conn) {
	var isReconnect bool

	defer func() {
		if !isReconnect {
			server.HandleDisconnection(conn)
		}
		conn.Close()
	}()

	buf := make([]byte, 1024)

	for {
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

		switch msg["type"] {
		case "register":
			if existingClient, ok := server.clients[msg["username"]]; ok && existingClient.Conn != nil {
				isReconnect = true
			}
			server.HandleRegistration(msg, conn)
		case "get_players":
			server.HandleGetPlayers(msg, conn)
		case "matchmake":
			server.HandleMatchmake(msg, conn)
		default:
			log.Println("Unknown message type:", msg["type"])
		}
	}
}
