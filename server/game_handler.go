package server

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func (server *Server) runGameLoop(player1, player2 *Client, squad1, squad2 []*battle.BattlePokemon, moveset1, moveset2 [][]*pokemon.MoveInfo) {
	battleState := NewBattleState(player1.Username, player2.Username, squad1, squad2)
	log.Printf("Starting game loop goroutine for player1=%s and player2=%s", player1.Username, player2.Username)

	server.mu.Lock()
	p1Client, p1Exists := server.clients[player1.Username]
	p2Client, p2Exists := server.clients[player2.Username]
	if !p1Exists || !p2Exists || p1Client.Conn == nil || p2Client.Conn == nil {
		server.mu.Unlock()
		return
	}
	lobby := &Lobby{player1: p1Client, player2: p2Client}
	server.Lobbies[player1.Username] = lobby
	server.Lobbies[player2.Username] = lobby
	log.Printf("Lobby confirmed/updated at start of runGameLoop for %s and %s", player1.Username, player2.Username)
	server.mu.Unlock()

	defer func() {}()

	for {
		p1Connected := player1.Conn != nil
		p2Connected := player2.Conn != nil
		if !p1Connected || !p2Connected {
			log.Printf("Player connection lost during game loop (%s:%v, %s:%v). Ending battle.", player1.Username, p1Connected, player2.Username, p2Connected)
			return
		}

		p1Moves := extractMoveNames(moveset1[battleState.Player1ActiveIndex])
		p2Moves := extractMoveNames(moveset2[battleState.Player2ActiveIndex])
		server.SendResponse(player1.Conn, Response{Type: "turn_request", Message: map[string]interface{}{"turn": battleState.TurnNumber, "available_moves": p1Moves}})
		server.SendResponse(player2.Conn, Response{Type: "turn_request", Message: map[string]interface{}{"turn": battleState.TurnNumber, "available_moves": p2Moves}})
		log.Printf("Turn %d: Sent turn requests to %s and %s", battleState.TurnNumber, player1.Username, player2.Username)

		actionResultChan1 := make(chan receivedAction)
		actionResultChan2 := make(chan receivedAction)
		go func() {
			action, err := receiveGameAction(player1.gameActionChan, player1.Username)
			actionResultChan1 <- receivedAction{action: action, err: err}
		}()
		go func() {
			action, err := receiveGameAction(player2.gameActionChan, player2.Username)
			actionResultChan2 <- receivedAction{action: action, err: err}
		}()

		var action1, action2 PlayerAction
		var err1, err2 error
		resultsReceived := 0
		for resultsReceived < 2 {
			select {
			case result1 := <-actionResultChan1:
				action1 = result1.action
				err1 = result1.err
				if err1 != nil {
					log.Printf("Turn %d: Error receiving action from %s: %v", battleState.TurnNumber, player1.Username, err1)
				} else {
					log.Printf("Turn %d: Received action from %s", battleState.TurnNumber, player1.Username)
				}
				resultsReceived++
			case result2 := <-actionResultChan2:
				action2 = result2.action
				err2 = result2.err
				if err2 != nil {
					log.Printf("Turn %d: Error receiving action from %s: %v", battleState.TurnNumber, player2.Username, err2)
				} else {
					log.Printf("Turn %d: Received action from %s", battleState.TurnNumber, player2.Username)
				}
				resultsReceived++
			}
		}

		if err1 != nil || err2 != nil {
			log.Printf("Turn %d: Errors detected during action receive (%v / %v), continuing outer loop for connection check.", battleState.TurnNumber, err1, err2)
			if err1 != nil && player1.Conn != nil {
				player1.Conn.Close()
				player1.Conn = nil
			}
			if err2 != nil && player2.Conn != nil {
				player2.Conn.Close()
				player2.Conn = nil
			}
			continue
		}

		log.Printf("Turn %d: Processing actions for %s and %s", battleState.TurnNumber, player1.Username, player2.Username)
		player1Pokemon := battleState.Player1Team[battleState.Player1ActiveIndex]
		player2Pokemon := battleState.Player2Team[battleState.Player2ActiveIndex]
		move1 := getMoveFromAction(action1, moveset1[battleState.Player1ActiveIndex])
		move2 := getMoveFromAction(action2, moveset2[battleState.Player2ActiveIndex])
		first, second, firstMove, secondMove := battle.ResolveTurn(player1Pokemon, player2Pokemon, move1, move2)
		turnSummary := []string{}
		var firstPlayerUsername, secondPlayerUsername string
		if first.UniqueID == player1Pokemon.UniqueID {
			firstPlayerUsername = player1.Username
			secondPlayerUsername = player2.Username
		} else {
			firstPlayerUsername = player2.Username
			secondPlayerUsername = player1.Username
		}

		if action1.Type == "switch" {
			if first.UniqueID == player1Pokemon.UniqueID {
				firstMove = nil
			} else {
				secondMove = nil
			}
		}
		if action2.Type == "switch" {
			if first.UniqueID == player2Pokemon.UniqueID {
				firstMove = nil
			} else {
				secondMove = nil
			}
		}
		if firstMove != nil && !first.Fainted {
			turnSummary = append(turnSummary, fmt.Sprintf("%s (%s) used %s!", first.Base.Name, firstPlayerUsername, firstMove.Name))
			if second.Fainted {
				turnSummary = append(turnSummary, fmt.Sprintf("%s (%s) fainted!", second.Base.Name, secondPlayerUsername))
			}
		} else if action1.Type != "switch" && action2.Type != "switch" {
		}
		if secondMove != nil && !second.Fainted {
			turnSummary = append(turnSummary, fmt.Sprintf("%s (%s) used %s!", second.Base.Name, secondPlayerUsername, secondMove.Name))
			if first.Fainted {
				turnSummary = append(turnSummary, fmt.Sprintf("%s (%s) fainted!", first.Base.Name, firstPlayerUsername))
			}
		} else if !second.Fainted && ((second.UniqueID == player1Pokemon.UniqueID && action1.Type == "move") || (second.UniqueID == player2Pokemon.UniqueID && action2.Type == "move")) {
		}
		if !player1Pokemon.Fainted {
			player1Pokemon.HandleTurnEffects()
		}
		if !player2Pokemon.Fainted {
			player2Pokemon.HandleTurnEffects()
		}
		turnSummary = append(turnSummary, "...End of turn effects applied...")

		log.Printf("Turn %d: Sending results to %s and %s", battleState.TurnNumber, player1.Username, player2.Username)
		battleState.LastTurnResults = turnSummary
		resultMsg := map[string]interface{}{"description": turnSummary}
		if player1.Conn != nil {
			server.SendResponse(player1.Conn, Response{Type: "turn_result", Message: resultMsg})
		}
		if player2.Conn != nil {
			server.SendResponse(player2.Conn, Response{Type: "turn_result", Message: resultMsg})
		}

		p1Lost := battle.IsAllFainted(battleState.Player1Team)
		p2Lost := battle.IsAllFainted(battleState.Player2Team)
		if p1Lost || p2Lost {
			log.Printf("Game over between %s and %s. P1 Lost: %v, P2 Lost: %v", player1.Username, player2.Username, p1Lost, p2Lost)
			winner, loser := player2, player1
			result := "win"
			if p1Lost && p2Lost {
				result = "draw"
			} else if p1Lost {
				winner, loser = player2, player1
				result = "win"
			}
			if winner.Conn != nil {
				server.sendGameEnd(winner, loser, result)
			}
			if loser.Conn != nil {
				server.sendGameEnd(loser, winner, mapResult(result))
			}
			return
		}

		battleState.TurnNumber++
	}
}

func (server *Server) sendGameEnd(playerToSendTo, opponent *Client, result string) {
	if playerToSendTo == nil || playerToSendTo.Conn == nil {
		return
	}
	message := "You lost the battle!"
	if result == "win" {
		message = "You won the battle!"
	}
	if result == "draw" {
		message = "The battle ended in a draw!"
	}
	server.SendResponse(playerToSendTo.Conn, Response{Type: "game_end", Message: map[string]interface{}{"result": result, "opponent": opponent.Username, "message": message}})
}

func mapResult(result string) string {
	if result == "win" {
		return "lose"
	}
	if result == "lose" {
		return "win"
	}
	return result
}

func extractMoveNames(moves []*pokemon.MoveInfo) []string {
	names := make([]string, 0, len(moves))
	for _, move := range moves {
		if move != nil {
			names = append(names, move.Name)
		} else {
			names = append(names, "(Invalid Move)")
		}
	}
	return names
}

type receivedAction struct {
	action PlayerAction
	err    error
}

func receiveGameAction(actionChan <-chan []byte, username string) (PlayerAction, error) {
	var action PlayerAction

	receiveTimeout := time.After(65 * time.Second)

	select {
	case data := <-actionChan:
		msgStr := strings.TrimSpace(string(data))
		log.Printf("receiveGameAction (%s): Received data from channel: %q", username, msgStr)

		if strings.HasPrefix(msgStr, "GAME_ACTION_MARKER|") {
			parts := strings.Split(msgStr, "|")
			if len(parts) == 4 {
				action.Type = parts[1]
				actionIdx, errIdx := strconv.Atoi(parts[2])
				switchIdx, errSwp := strconv.Atoi(parts[3])
				if errIdx != nil || errSwp != nil {
					return action, fmt.Errorf("invalid number format in action from %s: %s", username, msgStr)
				}
				if action.Type != "move" && action.Type != "switch" {
					return action, fmt.Errorf("invalid action type '%s' from %s", action.Type, username)
				}
				action.ActionIndex = actionIdx
				action.SwitchToIndex = switchIdx
				log.Printf("Parsed game action from %s: Type=%s, ActionIndex=%d, SwitchToIndex=%d", username, action.Type, action.ActionIndex, action.SwitchToIndex)
				return action, nil
			}
		}
		return action, fmt.Errorf("invalid game action format received from channel for %s: %s", username, msgStr)

	case <-receiveTimeout:
		return action, fmt.Errorf("timeout waiting for action from %s via channel", username)
	}
}

func getMoveFromAction(action PlayerAction, moves []*pokemon.MoveInfo) *pokemon.MoveInfo {
	// Validate action type
	if action.Type != "move" {
		return nil
	}

	// Validate action index
	if action.ActionIndex < 1 || action.ActionIndex > len(moves) {
		log.Printf("Invalid move index: %d (max: %d)", action.ActionIndex, len(moves))
		return nil
	}

	// Return the selected move
	return moves[action.ActionIndex-1]
}

