package server

import (
	"log"
	"net"
)

func (server *Server) handleRegistration(msg map[string]string, conn net.Conn) {
	username := msg["username"]
	if username == "" {
		log.Println("Username cannot be empty")
		return
	}

	if _, exists := server.clients[username]; exists {
		log.Printf("Username %s is already taken", username)
		return
	}

	client := &Client{conn: conn, username: username}
	server.clients[username] = client
	log.Printf("Player %s registered successfully", username)

	response := Response{
		Type: "registration_success",
		Message: map[string]interface{}{
			"username": username,
		},
	}
	server.SendResponse(conn, response)
}

func (server *Server) handleGetPlayers(conn net.Conn) {
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

func (server *Server) handleMatchmake(msg map[string]string, conn net.Conn) {
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

	if server.isInLobby(opponentClient) {
		log.Printf("Opponent %s is already in a match", opponent)
		return
	}

	lobby := &Lobby{player1: player, player2: opponentClient}
	server.lobbies[username] = lobby
	server.lobbies[opponent] = lobby

	log.Printf("Match started between %s and %s", username, opponent)

	server.SendResponse(player.conn, Response{
		Type: "match_start",
		Message: map[string]interface{}{
			"opponent": opponent,
		},
	})
	server.SendResponse(opponentClient.conn, Response{
		Type: "match_start",
		Message: map[string]interface{}{
			"opponent": username,
		},
	})

	server.startGame(lobby)
}

func (server *Server) isInLobby(client *Client) bool {
	for _, lobby := range server.lobbies {
		if lobby.player1 == client || lobby.player2 == client {
			return true
		}
	}
	return false
}

func (server *Server) startGame(lobby *Lobby) {
	log.Println("Game started between", lobby.player1.username, "and", lobby.player2.username)

	server.SendResponse(lobby.player1.conn, Response{
		Type: "game_end",
		Message: map[string]interface{}{
			"result": "win",
		},
	})
	server.SendResponse(lobby.player2.conn, Response{
		Type: "game_end",
		Message: map[string]interface{}{
			"result": "lose",
		},
	})

	delete(server.lobbies, lobby.player1.username)
	delete(server.lobbies, lobby.player2.username)
}

