package server

import (
	"fmt"
	"log"
	"net"
)

func (server *Server) HandleRegistration(msg map[string]string, conn net.Conn) {
	username := msg["username"]
	if username == "" {
		log.Println("Username cannot be empty")
		return
	}

	if existingClient, ok := server.clients[username]; ok {
		if existingClient.Conn != nil {
			log.Printf("Player %s is already connected. Disconnecting the previous client...", username)
			existingClient.Conn.Close()
			log.Printf("Disconnected previous connection of player %s", username)
		}
		existingClient.Conn = conn
		log.Printf("Player %s reconnected", username)

		server.SendResponse(conn, Response{
			Type: "reconnect",
			Message: map[string]interface{}{
				"username": username,
				"status":   "Reconnected successfully",
			},
		})
		return
	}

	newClient := &Client{Username: username, Conn: conn}
	server.clients[username] = newClient
	log.Printf("Player %s registered successfully", username)

	server.SendResponse(conn, Response{
		Type: "registration",
		Message: map[string]interface{}{
			"username": username,
			"status":   fmt.Sprintf("Player %s registered successfully", username),
		},
	})
}

func (server *Server) HandleGetPlayers(conn net.Conn) {
	var players []string
	for username := range server.clients {
		players = append(players, username)
	}

	response := Response{
		Type: "player_list",
		Message: map[string]interface{}{
			"players": players,
		},
	}

	server.SendResponse(conn, response)
}

func (server *Server) HandleMatchmake(msg map[string]string, conn net.Conn) {
	username := msg["username"]
	opponent := msg["opponent"]

	player, exists := server.clients[username]
	if !exists {
		log.Printf("Player %s not found", username)
		return
	}
	opponentClient, exists := server.clients[opponent]
	if !exists {
		log.Printf("Opponent %s not found", opponent)
		return
	}

	if server.IsInLobby(player) {
		log.Printf("Player %s is already in a match", username)
		return
	}

	if server.IsInLobby(opponentClient) {
		log.Printf("Opponent %s is already in a match", opponent)
		return
	}

	lobby := &Lobby{player1: player, player2: opponentClient}
	server.Lobbies[username] = lobby
	server.Lobbies[opponent] = lobby

	log.Printf("Match started between %s and %s", username, opponent)

	server.SendResponse(player.Conn, Response{
		Type: "match_start",
		Message: map[string]interface{}{
			"opponent": opponent,
		},
	})
	server.SendResponse(opponentClient.Conn, Response{
		Type: "match_start",
		Message: map[string]interface{}{
			"opponent": username,
		},
	})

	server.startGame(lobby)
}

func (server *Server) HandleDisconnection(conn net.Conn) {
	for username, client := range server.clients {
		if client.Conn == conn {
			log.Printf("Player %s disconnected", username)

			delete(server.clients, username)

			for key, lobby := range server.Lobbies {
				if lobby.player1.Username == username || lobby.player2.Username == username {
					delete(server.Lobbies, key)
				}
			}
			return
		}
	}
}

func (server *Server) HandleReconnection(msg map[string]string, conn net.Conn) {
	username := msg["username"]

	if existingClient, ok := server.clients[username]; ok {
		if existingClient.Conn != nil {
			log.Printf("Player %s is already connected", username)
			existingClient.Conn.Close()
		}
	}

	newClient := &Client{Username: username, Conn: conn}
	server.clients[username] = newClient
}

func (server *Server) IsInLobby(client *Client) bool {
	_, exists := server.Lobbies[client.Username]
	return exists
}

func (server *Server) startGame(lobby *Lobby) {
	log.Println("Game started between", lobby.player1.Username, "and", lobby.player2.Username)

	server.SendResponse(lobby.player1.Conn, Response{
		Type: "game_end",
		Message: map[string]interface{}{
			"result": "win",
		},
	})

	server.SendResponse(lobby.player2.Conn, Response{
		Type: "game_end",
		Message: map[string]interface{}{
			"result": "lose",
		},
	})

	delete(server.Lobbies, lobby.player1.Username)
	delete(server.Lobbies, lobby.player2.Username)
}
