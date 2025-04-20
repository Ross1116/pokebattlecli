package server

import (
	"fmt"
	"log"
	"net"

	"github.com/ross1116/pokebattlecli/internal/battle"
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

func (server *Server) HandleGetPlayers(msg map[string]string, conn net.Conn) {
	requestingUsername := msg["username"]
	log.Printf("Get players request from: %s", requestingUsername)

	var players []string
	log.Printf("Total clients in map: %d", len(server.clients))

	for username, client := range server.clients {
		log.Printf("Checking client: %s, conn nil: %v", username, client.Conn == nil)
		if client.Conn != nil {
			players = append(players, username)
		}
	}

	log.Printf("Returning player list with %d players: %v", len(players), players)

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

	if username == opponent {
		server.SendResponse(conn, Response{
			Type:    "match_error",
			Message: map[string]interface{}{"error": "Cannot match with yourself"},
		})
		return
	}

	player, exists := server.clients[username]
	if !exists {
		log.Printf("Player %s not found", username)
		return
	}

	opponentClient, exists := server.clients[opponent]
	if !exists {
		log.Printf("Opponent %s not found", opponent)
		server.SendResponse(conn, Response{
			Type:    "match_error",
			Message: map[string]interface{}{"error": "Opponent not found"},
		})
		return
	}

	if server.IsInLobby(player) {
		log.Printf("Player %s is already in a match", username)
		server.SendResponse(conn, Response{
			Type:    "match_error",
			Message: map[string]interface{}{"error": "You are already in a match"},
		})
		return
	}

	if server.IsInLobby(opponentClient) {
		log.Printf("Opponent %s is already in a match", opponent)
		server.SendResponse(conn, Response{
			Type:    "match_error",
			Message: map[string]interface{}{"error": "Opponent is already in a match"},
		})
		return
	}

	if opponentClient.Conn == nil {
		log.Printf("Opponent %s has disconnected", opponent)
		server.SendResponse(conn, Response{
			Type:    "match_error",
			Message: map[string]interface{}{"error": "Opponent has disconnected"},
		})
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

			if lobby, inLobby := server.Lobbies[username]; inLobby {
				var opponentUsername string
				if lobby.player1.Username == username {
					opponentUsername = lobby.player2.Username
				} else {
					opponentUsername = lobby.player1.Username
				}

				if opponent, exists := server.clients[opponentUsername]; exists && opponent.Conn != nil {
					server.SendResponse(opponent.Conn, Response{
						Type: "opponent_disconnected",
						Message: map[string]interface{}{
							"opponent": username,
						},
					})
				}

				delete(server.Lobbies, username)
				delete(server.Lobbies, opponentUsername)

				log.Printf("Removed match between %s and %s due to disconnection",
					username, opponentUsername)
			}

			client.Conn = nil
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

	squad1, squad2, moveset1, moveset2, idx1, idx2 := battle.SetupMPSquad()

	squad1Names := []string{}
	for _, poke := range squad1 {
		squad1Names = append(squad1Names, poke.Base.Name)
	}

	squad2Names := []string{}
	for _, poke := range squad2 {
		squad2Names = append(squad2Names, poke.Base.Name)
	}

	moveNames1 := make([]string, len(moveset1[idx1]))
	for i, move := range moveset1[idx1] {
		moveNames1[i] = move.Name
	}

	moveNames2 := make([]string, len(moveset2[idx2]))
	for i, move := range moveset2[idx2] {
		moveNames2[i] = move.Name
	}

	server.SendResponse(lobby.player1.Conn, Response{
		Type: "game_start",
		Message: map[string]interface{}{
			"your_squad":       squad1Names,
			"opponent_squad":   squad2Names,
			"your_pokemon":     squad1Names[idx1],
			"opponent_pokemon": squad2Names[idx2],
			"your_moves":       moveNames1,
		},
	})

	server.SendResponse(lobby.player2.Conn, Response{
		Type: "game_start",
		Message: map[string]interface{}{
			"your_squad":       squad2Names,
			"opponent_squad":   squad1Names,
			"your_pokemon":     squad2Names[idx2],
			"opponent_pokemon": squad1Names[idx1],
			"your_moves":       moveNames2,
		},
	})

	delete(server.Lobbies, lobby.player1.Username)
	delete(server.Lobbies, lobby.player2.Username)
}
