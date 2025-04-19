package server

import (
	"encoding/json"
	"log"
	"net"
)

func (server *Server) handleClient(conn net.Conn) {
	defer conn.Close()

	// Read incoming message
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Failed to read from client:", err)
		return
	}

	var msg map[string]string
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		log.Println("Invalid message from client:", err)
		return
	}

	switch msg["type"] {
	case "register":
		server.handleRegistration(msg, conn)
	case "join_lobby":
		server.handleJoinLobby(msg, conn)
	case "battle":
		server.handleBattle(msg, conn)
	default:
		log.Println("Unknown message type:", msg["type"])
	}
}

func (server *Server) handleRegistration(msg map[string]string, conn net.Conn) {
	username := msg["username"]
	if username == "" {
		log.Println("Username must not be empty")
		return
	}

	server.clients[username] = &Client{conn: conn, username: username}

	log.Printf("Player %s registered successfully", username)

	response := map[string]string{"type": "registration_success", "username": username}
	server.SendResponse(conn, response)
}

func (server *Server) handleJoinLobby(msg map[string]string, conn net.Conn) {

}

func (server *Server) handleBattle(msg map[string]string, conn net.Conn) {

}
