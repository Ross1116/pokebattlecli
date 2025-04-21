package server

import (
	"fmt"
	"log"
	"net"

	"github.com/ross1116/pokebattlecli/internal/battle"
)

func (server *Server) HandleRegistration(msg map[string]string, conn net.Conn) {
	username := msg["username"]

	server.mu.RLock()
	_, clientAlreadyExisted := server.clients[username]
	server.mu.RUnlock()

	registrationStatus := "registered/reconnected"
	if clientAlreadyExisted {
		log.Printf("HandleRegistration: Confirmed client %s exists (likely reconnected).", username)
	} else {
		log.Printf("HandleRegistration: Client %s was not found in map (should have been added by HandleClient).", username)
		registrationStatus = "registered"
	}

	server.SendResponse(conn, Response{
		Type: "registration",
		Message: map[string]interface{}{
			"username": username,
			"status":   fmt.Sprintf("Player %s %s successfully", username, registrationStatus),
		},
	})
}

func (server *Server) HandleGetPlayers(msg map[string]string, conn net.Conn) {
	server.mu.RLock()
	defer server.mu.RUnlock()
	var players []string
	for username, client := range server.clients {
		if client.Conn != nil {
			players = append(players, username)
		}
	}
	log.Printf("Returning player list: %v", players)
	response := Response{Type: "player_list", Message: map[string]interface{}{"players": players}}
	server.SendResponse(conn, response)
}

func (server *Server) HandleMatchmake(msg map[string]string, conn net.Conn) {
	username := msg["username"]
	opponentName := msg["opponent"]
	if username == "" {
		return
	}
	if username == opponentName {
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Cannot match with yourself"}})
		return
	}

	server.mu.RLock()
	player, playerExists := server.clients[username]
	opponentClient, opponentExists := server.clients[opponentName]
	_, playerInLobby := server.Lobbies[username]
	_, opponentInLobby := server.Lobbies[opponentName]
	opponentConnValid := opponentExists && opponentClient.Conn != nil
	server.mu.RUnlock()

	if !playerExists {
		log.Printf("Matchmake Error: Requesting user %s not found.", username)
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Internal server error (player not found)"}})
		return
	}
	if !opponentExists {
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Opponent not found"}})
		return
	}
	if playerInLobby {
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "You are already in a match"}})
		return
	}
	if opponentInLobby {
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Opponent is already in a match"}})
		return
	}
	if !opponentConnValid {
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Opponent has disconnected"}})
		return
	}

	server.mu.Lock()
	if _, stillInLobby1 := server.Lobbies[username]; stillInLobby1 {
		server.mu.Unlock()
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Race condition: You were matched just now"}})
		return
	}
	if _, stillInLobby2 := server.Lobbies[opponentName]; stillInLobby2 {
		server.mu.Unlock()
		server.SendResponse(conn, Response{Type: "match_error", Message: map[string]interface{}{"error": "Race condition: Opponent was matched just now"}})
		return
	}
	lobby := &Lobby{player1: player, player2: opponentClient}
	server.Lobbies[username] = lobby
	server.Lobbies[opponentName] = lobby
	log.Printf("Lobby created and stored for %s and %s", username, opponentName)
	server.mu.Unlock()

	log.Printf("Match successfully initiated between %s and %s", username, opponentName)
	server.SendResponse(player.Conn, Response{Type: "match_start", Message: map[string]interface{}{"opponent": opponentName}})
	server.SendResponse(opponentClient.Conn, Response{Type: "match_start", Message: map[string]interface{}{"opponent": username}})
	go server.startGame(player, opponentClient)
}

func (server *Server) HandleDisconnection(conn net.Conn, username string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	disconnectedUser := ""
	if username != "" {
		disconnectedUser = username
	} else {
		for name, client := range server.clients {
			if client.Conn == conn {
				disconnectedUser = name
				break
			}
		}
	}
	if disconnectedUser == "" {
		return
	}

	client, exists := server.clients[disconnectedUser]
	if !exists {
		return
	}

	log.Printf("Handling disconnection for player %s (%s)", disconnectedUser, conn.RemoteAddr())

	if client.Conn == conn {
		client.Conn = nil
	}

	if lobby, inLobby := server.Lobbies[disconnectedUser]; inLobby {
		var opponentUsername string
		var opponentClient *Client
		if lobby.player1 != nil && lobby.player1.Username == disconnectedUser {
			if lobby.player2 != nil {
				opponentUsername = lobby.player2.Username
				opponentClient = lobby.player2
			}
		} else if lobby.player2 != nil && lobby.player2.Username == disconnectedUser {
			if lobby.player1 != nil {
				opponentUsername = lobby.player1.Username
				opponentClient = lobby.player1
			}
		}

		delete(server.Lobbies, disconnectedUser)
		if opponentUsername != "" {
			delete(server.Lobbies, opponentUsername)
		}
		log.Printf("Removed lobby involving %s due to disconnection.", disconnectedUser)

		if opponentClient != nil {
			if opponentClient.endGameSignal != nil {
				select {
				case <-opponentClient.endGameSignal:
				default:
					close(opponentClient.endGameSignal)
				}
			}
			if opponentClient.Conn != nil {
				opponentConn := opponentClient.Conn
				go func() {
					log.Printf("Notifying %s about %s's disconnection.", opponentUsername, disconnectedUser)
					server.SendResponse(opponentConn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": disconnectedUser}})
				}()
			}
		}
		if client.startGameSignal != nil {
			select {
			case <-client.startGameSignal:
			default:
				close(client.startGameSignal)
			}
		}
		if client.endGameSignal != nil {
			select {
			case <-client.endGameSignal:
			default:
				close(client.endGameSignal)
			}
		}

	} else {
		log.Printf("Player %s disconnected (was not in a lobby).", disconnectedUser)
	}
}

func (server *Server) startGame(player1, player2 *Client) {
	if player1 == nil || player2 == nil {
		log.Println("startGame Error: Invalid client(s) provided.")
		return
	}

	log.Printf("startGame invoked for %s and %s", player1.Username, player2.Username)

	squad1, squad2, moveset1, moveset2, idx1, idx2 := battle.SetupMPSquad()
	log.Printf("Squads generated for %s and %s", player1.Username, player2.Username)

	squad1Names := make([]string, len(squad1))
	for i, p := range squad1 {
		squad1Names[i] = p.Base.Name
	}
	squad2Names := make([]string, len(squad2))
	for i, p := range squad2 {
		squad2Names[i] = p.Base.Name
	}
	moveNames1 := make([]string, len(moveset1[idx1]))
	for i, m := range moveset1[idx1] {
		moveNames1[i] = m.Name
	}
	moveNames2 := make([]string, len(moveset2[idx2]))
	for i, m := range moveset2[idx2] {
		moveNames2[i] = m.Name
	}

	if player1.Conn != nil {
		server.SendResponse(player1.Conn, Response{Type: "game_start", Message: map[string]interface{}{"your_squad": squad1Names, "opponent_squad": squad2Names, "your_pokemon": squad1Names[idx1], "opponent_pokemon": squad2Names[idx2], "your_moves": moveNames1}})
	}
	if player2.Conn != nil {
		server.SendResponse(player2.Conn, Response{Type: "game_start", Message: map[string]interface{}{"your_squad": squad2Names, "opponent_squad": squad1Names, "your_pokemon": squad2Names[idx2], "opponent_pokemon": squad1Names[idx1], "your_moves": moveNames2}})
	}

	if player1.startGameSignal != nil {
		select {
		case <-player1.startGameSignal:
		default:
			close(player1.startGameSignal)
		}
	}
	if player2.startGameSignal != nil {
		select {
		case <-player2.startGameSignal:
		default:
			close(player2.startGameSignal)
		}
	}

	log.Printf("game_start messages sent and signals sent to %s and %s. Starting runGameLoop.", player1.Username, player2.Username)

	server.runGameLoop(player1, player2, squad1, squad2, moveset1, moveset2)

	log.Printf("startGame finished for lobby between %s and %s", player1.Username, player2.Username)
}

