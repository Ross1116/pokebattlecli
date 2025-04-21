package server

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func (server *Server) runGameLoop(player1, player2 *Client, squad1, squad2 []*battle.BattlePokemon, moveset1, moveset2 [][]*pokemon.MoveInfo) {
	battleState := NewBattleState(player1.Username, player2.Username, squad1, squad2)

	for {
		server.SendResponse(player1.Conn, Response{
			Type: "turn_request",
			Message: map[string]interface{}{
				"turn":            battleState.TurnNumber,
				"available_moves": extractMoveNames(moveset1[battleState.Player1ActiveIndex]),
			},
		})
		server.SendResponse(player2.Conn, Response{
			Type: "turn_request",
			Message: map[string]interface{}{
				"turn":            battleState.TurnNumber,
				"available_moves": extractMoveNames(moveset2[battleState.Player2ActiveIndex]),
			},
		})

		action1 := receivePlayerAction(player1.Conn)
		action2 := receivePlayerAction(player2.Conn)

		player1Pokemon := battleState.Player1Team[battleState.Player1ActiveIndex]
		player2Pokemon := battleState.Player2Team[battleState.Player2ActiveIndex]

		move1 := getMoveFromAction(action1, moveset1[battleState.Player1ActiveIndex])
		move2 := getMoveFromAction(action2, moveset2[battleState.Player2ActiveIndex])

		first, second, firstMove, secondMove := battle.ResolveTurn(player1Pokemon, player2Pokemon, move1, move2)

		var turnSummary []string

		if action1.Type == "switch" && first == player1Pokemon {
			battleState.Player1ActiveIndex = action1.SwitchToIndex
			turnSummary = append(turnSummary, fmt.Sprintf("%s switched to %s", player1.Username, battleState.Player1Team[action1.SwitchToIndex].Base.Name))
		} else if action2.Type == "switch" && first == player2Pokemon {
			battleState.Player2ActiveIndex = action2.SwitchToIndex
			turnSummary = append(turnSummary, fmt.Sprintf("%s switched to %s", player2.Username, battleState.Player2Team[action2.SwitchToIndex].Base.Name))
		} else {
			battle.ProcessPlayerTurn(first, second, firstMove)
			turnSummary = append(turnSummary, fmt.Sprintf("%s used %s", first.Base.Name, firstMove.Name))
		}

		if second.Fainted {
			turnSummary = append(turnSummary, fmt.Sprintf("%s fainted!", second.Base.Name))
		} else {
			if action2.Type == "switch" && second == player2Pokemon {
				battleState.Player2ActiveIndex = action2.SwitchToIndex
				turnSummary = append(turnSummary, fmt.Sprintf("%s switched to %s", player2.Username, battleState.Player2Team[action2.SwitchToIndex].Base.Name))
			} else if action1.Type == "switch" && second == player1Pokemon {
				battleState.Player1ActiveIndex = action1.SwitchToIndex
				turnSummary = append(turnSummary, fmt.Sprintf("%s switched to %s", player1.Username, battleState.Player1Team[action1.SwitchToIndex].Base.Name))
			} else {
				battle.ProcessPlayerTurn(second, first, secondMove)
				turnSummary = append(turnSummary, fmt.Sprintf("%s used %s", second.Base.Name, secondMove.Name))
			}
			if first.Fainted {
				turnSummary = append(turnSummary, fmt.Sprintf("%s fainted!", first.Base.Name))
			}
		}

		player1Pokemon.HandleTurnEffects()
		player2Pokemon.HandleTurnEffects()

		battleState.LastTurnResults = turnSummary
		battleState.TurnNumber++

		server.SendResponse(player1.Conn, Response{
			Type:    "turn_result",
			Message: map[string]interface{}{"description": turnSummary},
		})
		server.SendResponse(player2.Conn, Response{
			Type:    "turn_result",
			Message: map[string]interface{}{"description": turnSummary},
		})

		if battle.IsAllFainted(battleState.Player1Team) {
			server.sendGameEnd(player2, player1)
			break
		}
		if battle.IsAllFainted(battleState.Player2Team) {
			server.sendGameEnd(player1, player2)
			break
		}
	}
}

func (server *Server) sendGameEnd(winner, loser *Client) {
	server.SendResponse(winner.Conn, Response{
		Type: "game_end",
		Message: map[string]interface{}{
			"result":  "win",
			"message": "You won the battle!",
		},
	})
	server.SendResponse(loser.Conn, Response{
		Type: "game_end",
		Message: map[string]interface{}{
			"result":  "lose",
			"message": "You lost the battle!",
		},
	})
}

func extractMoveNames(moves []*pokemon.MoveInfo) []string {
	names := []string{}
	for _, move := range moves {
		names = append(names, move.Name)
	}
	return names
}

func receivePlayerAction(conn net.Conn) PlayerAction {
	var action PlayerAction
	decoder := json.NewDecoder(conn)
	err := decoder.Decode(&action)
	if err != nil {
		fmt.Println("Failed to decode player action:", err)
	}
	return action
}

func getMoveFromAction(action PlayerAction, moves []*pokemon.MoveInfo) *pokemon.MoveInfo {
	if action.Type == "move" && action.ActionIndex >= 1 && action.ActionIndex <= len(moves) {
		return moves[action.ActionIndex-1]
	}
	return nil
}
