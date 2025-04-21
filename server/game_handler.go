package server

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

type PokemonStateInfo struct {
	SquadIndex int     `json:"squad_index"`
	Name       string  `json:"name"`
	CurrentHP  float64 `json:"current_hp"`
	MaxHP      float64 `json:"max_hp"`
	HPPercent  float64 `json:"hp_percent"`
	Fainted    bool    `json:"fainted"`
	Status     string  `json:"status"`
}

func getSquadStateInfo(squad []*battle.BattlePokemon) []PokemonStateInfo {
	if squad == nil {
		return nil
	}
	info := make([]PokemonStateInfo, len(squad))
	for i, p := range squad {
		if p == nil || p.Base == nil {
			info[i] = PokemonStateInfo{SquadIndex: i, Name: "(Error)"}
			continue
		}
		maxHP := 0.0
		hpBaseStat := stats.GetStat(p.Base, "hp")
		if hpBaseStat > 0 {
			maxHP = stats.HpCalc(hpBaseStat)
		} else {
			log.Printf("Warning: Could not get HP base stat for %s", p.Base.Name)
		}
		hpPercent := 0.0
		if maxHP > 0 {
			hpPercent = math.Max(0, math.Min(100, (p.CurrentHP/maxHP)*100.0))
		}
		info[i] = PokemonStateInfo{
			SquadIndex: i, Name: p.Base.Name, CurrentHP: p.CurrentHP, MaxHP: maxHP,
			HPPercent: hpPercent, Fainted: p.Fainted, Status: p.Status,
		}
	}
	return info
}

func (server *Server) runGameLoop(player1, player2 *Client, squad1, squad2 []*battle.BattlePokemon, moveset1, moveset2 [][]*pokemon.MoveInfo) {
	battleState := NewBattleState(player1.Username, player2.Username, squad1, squad2)
	log.Printf("Starting game loop goroutine for player1=%s and player2=%s", player1.Username, player2.Username)

	server.mu.Lock()
	lobby, lobbyExists := server.Lobbies[player1.Username]
	if !lobbyExists {
		lobby = &Lobby{player1: player1, player2: player2}
		server.Lobbies[player1.Username] = lobby
		server.Lobbies[player2.Username] = lobby
		log.Printf("Lobby created/re-added at start of runGameLoop for %s and %s", player1.Username, player2.Username)
	} else {
		lobby.player1 = player1
		lobby.player2 = player2
	}
	server.mu.Unlock()

	defer func() {
		log.Printf("runGameLoop ending for %s and %s. Cleaning up lobby and signaling.", player1.Username, player2.Username)
		server.mu.Lock()
		delete(server.Lobbies, player1.Username)
		delete(server.Lobbies, player2.Username)
		log.Printf("Lobby removed via defer in runGameLoop for %s and %s", player1.Username, player2.Username)
		server.mu.Unlock()
		if player1.endGameSignal != nil {
			select {
			case <-player1.endGameSignal:
			default:
				close(player1.endGameSignal)
			}
		}
		if player2.endGameSignal != nil {
			select {
			case <-player2.endGameSignal:
			default:
				close(player2.endGameSignal)
			}
		}
		log.Printf("Game end signaled to HandleClients for %s and %s", player1.Username, player2.Username)
	}()

	for {
		p1Connected := player1.Conn != nil
		p2Connected := player2.Conn != nil
		if !p1Connected || !p2Connected {
			log.Printf("Player connection lost during game loop (%s:%v, %s:%v). Ending battle.", player1.Username, p1Connected, player2.Username, p2Connected)
			if !p1Connected && p2Connected && player2.Conn != nil {
				server.SendResponse(player2.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player1.Username}})
			}
			if !p2Connected && p1Connected && player1.Conn != nil {
				server.SendResponse(player1.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player2.Username}})
			}
			return
		}

		p1Lost := battle.IsAllFainted(battleState.Player1Team)
		p2Lost := battle.IsAllFainted(battleState.Player2Team)
		if p1Lost || p2Lost {
			log.Printf("Game over detected at start of Turn %d. P1 Lost: %v, P2 Lost: %v", battleState.TurnNumber, p1Lost, p2Lost)
			resultP1, resultP2 := "lose", "win"
			if p1Lost && p2Lost {
				resultP1, resultP2 = "draw", "draw"
			}
			if p1Lost {
				resultP1, resultP2 = "lose", "win"
			}
			if p2Lost {
				resultP1, resultP2 = "win", "lose"
			}
			if player1.Conn != nil {
				server.sendGameEnd(player1, player2, resultP1)
			}
			if player2.Conn != nil {
				server.sendGameEnd(player2, player1, resultP2)
			}
			return
		}

		p1MustSwitchAtTurnStart := battleState.Player1Team[battleState.Player1ActiveIndex].Fainted
		p2MustSwitchAtTurnStart := battleState.Player2Team[battleState.Player2ActiveIndex].Fainted
		log.Printf("Turn %d: Start of turn faint check: P1 Must Switch: %t, P2 Must Switch: %t", battleState.TurnNumber, p1MustSwitchAtTurnStart, p2MustSwitchAtTurnStart)

		if p1MustSwitchAtTurnStart {
			server.SendResponse(player1.Conn, Response{Type: "turn_request", Message: map[string]interface{}{
				"turn": battleState.TurnNumber, "available_moves": []string{}, "force_switch": true,
			}})
		} else {
			p1Moves := extractMoveNames(moveset1[battleState.Player1ActiveIndex])
			server.SendResponse(player1.Conn, Response{Type: "turn_request", Message: map[string]interface{}{
				"turn": battleState.TurnNumber, "available_moves": p1Moves, "force_switch": false,
			}})
		}
		if p2MustSwitchAtTurnStart {
			server.SendResponse(player2.Conn, Response{Type: "turn_request", Message: map[string]interface{}{
				"turn": battleState.TurnNumber, "available_moves": []string{}, "force_switch": true,
			}})
		} else {
			p2Moves := extractMoveNames(moveset2[battleState.Player2ActiveIndex])
			server.SendResponse(player2.Conn, Response{Type: "turn_request", Message: map[string]interface{}{
				"turn": battleState.TurnNumber, "available_moves": p2Moves, "force_switch": false,
			}})
		}
		log.Printf("Turn %d: Sent turn requests to %s and %s", battleState.TurnNumber, player1.Username, player2.Username)

		actionResultChan1 := make(chan receivedAction)
		actionResultChan2 := make(chan receivedAction)
		go func() {
			actionResultChan1 <- receiveGameAction(player1.gameActionChan, player1.Username, GameActionMarker)
		}()
		go func() {
			actionResultChan2 <- receiveGameAction(player2.gameActionChan, player2.Username, GameActionMarker)
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
					log.Printf("Turn %d: Error/Timeout receiving action from %s: %v", battleState.TurnNumber, player1.Username, err1)
				}
				resultsReceived++
			case result2 := <-actionResultChan2:
				action2 = result2.action
				err2 = result2.err
				if err2 != nil {
					log.Printf("Turn %d: Error/Timeout receiving action from %s: %v", battleState.TurnNumber, player2.Username, err2)
				}
				resultsReceived++
			}
		}
		if err1 != nil || err2 != nil {
			log.Printf("Turn %d: Errors/disconnects during action receive (%s:%v / %s:%v), ending game.", battleState.TurnNumber, player1.Username, err1, player2.Username, err2)
			if err1 != nil && player2.Conn != nil {
				server.SendResponse(player2.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player1.Username, "reason": "Timeout/Error"}})
			}
			if err2 != nil && player1.Conn != nil {
				server.SendResponse(player1.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player2.Username, "reason": "Timeout/Error"}})
			}
			if player1.Conn != nil {
				player1.Conn.Close()
				player1.Conn = nil
			}
			if player2.Conn != nil {
				player2.Conn.Close()
				player2.Conn = nil
			}
			return
		}
		log.Printf("Turn %d: Received actions: P1=%+v, P2=%+v", battleState.TurnNumber, action1, action2)

		if p1MustSwitchAtTurnStart && action1.Type != "switch" {
			log.Printf("Turn %d: Player 1 %s was forced to switch but sent action type '%s'. Invalidating action.", battleState.TurnNumber, player1.Username, action1.Type)
			action1 = PlayerAction{Type: "switch", SwitchToIndex: -1}
		}
		if p2MustSwitchAtTurnStart && action2.Type != "switch" {
			log.Printf("Turn %d: Player 2 %s was forced to switch but sent action type '%s'. Invalidating action.", battleState.TurnNumber, player2.Username, action2.Type)
			action2 = PlayerAction{Type: "switch", SwitchToIndex: -1}
		}

		log.Printf("Turn %d: Processing actions for %s and %s", battleState.TurnNumber, player1.Username, player2.Username)
		turnSummary := []string{}
		var player1Move, player2Move *pokemon.MoveInfo
		p1Switched := false
		p2Switched := false

		if action1.Type == "switch" {
			targetIdx := action1.SwitchToIndex
			if targetIdx >= 0 && targetIdx < len(battleState.Player1Team) && !battleState.Player1Team[targetIdx].Fainted && targetIdx != battleState.Player1ActiveIndex {
				battleState.Player1ActiveIndex = targetIdx
				turnSummary = append(turnSummary, fmt.Sprintf("%s switched to %s!", player1.Username, battleState.Player1Team[targetIdx].Base.Name))
				p1Switched = true
			} else {
				errMsg := fmt.Sprintf("%s tried to switch but failed!", player1.Username)
				if p1MustSwitchAtTurnStart {
					errMsg = fmt.Sprintf("%s failed to select a valid Pokemon to switch to!", player1.Username)
				}
				turnSummary = append(turnSummary, errMsg)
			}
			player1Move = nil
		}
		if action2.Type == "switch" {
			targetIdx := action2.SwitchToIndex
			if targetIdx >= 0 && targetIdx < len(battleState.Player2Team) && !battleState.Player2Team[targetIdx].Fainted && targetIdx != battleState.Player2ActiveIndex {
				battleState.Player2ActiveIndex = targetIdx
				turnSummary = append(turnSummary, fmt.Sprintf("%s switched to %s!", player2.Username, battleState.Player2Team[targetIdx].Base.Name))
				p2Switched = true
			} else {
				errMsg := fmt.Sprintf("%s tried to switch but failed!", player2.Username)
				if p2MustSwitchAtTurnStart {
					errMsg = fmt.Sprintf("%s failed to select a valid Pokemon to switch to!", player2.Username)
				}
				turnSummary = append(turnSummary, errMsg)
			}
			player2Move = nil
		}

		if !p1Switched && action1.Type == "move" {
			player1Move = getMoveFromAction(action1, moveset1[battleState.Player1ActiveIndex])
			if player1Move == nil {
				turnSummary = append(turnSummary, fmt.Sprintf("%s failed to select a valid move!", player1.Username))
			}
		}
		if !p2Switched && action2.Type == "move" {
			player2Move = getMoveFromAction(action2, moveset2[battleState.Player2ActiveIndex])
			if player2Move == nil {
				turnSummary = append(turnSummary, fmt.Sprintf("%s failed to select a valid move!", player2.Username))
			}
		}

		p1FinalActing := battleState.Player1Team[battleState.Player1ActiveIndex]
		p2FinalActing := battleState.Player2Team[battleState.Player2ActiveIndex]
		if player1Move != nil || player2Move != nil {
			battleEvents := battle.ExecuteBattleTurn(p1FinalActing, p2FinalActing, player1Move, player2Move)
			turnSummary = append(turnSummary, battleEvents...)
		} else {
			if len(turnSummary) == 0 {
				turnSummary = append(turnSummary, "Neither Pokemon could make a move!")
			}
		}

		p1FaintedThisTurn := p1FinalActing.Fainted
		p2FaintedThisTurn := p2FinalActing.Fainted
		log.Printf("Turn %d: Mid-turn faint check: P1 Fainted: %t, P2 Fainted: %t", battleState.TurnNumber, p1FaintedThisTurn, p2FaintedThisTurn)

		if p1FaintedThisTurn && !battle.IsAllFainted(battleState.Player1Team) {
			log.Printf("Turn %d: Player 1's %s fainted mid-turn. Requesting switch.", battleState.TurnNumber, p1FinalActing.Base.Name)
			server.SendResponse(player1.Conn, Response{Type: "switch_request", Message: map[string]interface{}{
				"reason": "Pokemon fainted",
			}})
			switchAction1, switchErr1 := receiveSwitchAction(player1.gameActionChan, player1.Username)
			if switchErr1 != nil {
				log.Printf("Turn %d: Error receiving switch action from %s: %v. Ending game.", battleState.TurnNumber, player1.Username, switchErr1)
				if player2.Conn != nil {
					server.SendResponse(player2.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player1.Username, "reason": "Switch Timeout/Error"}})
				}
				if player1.Conn != nil {
					player1.Conn.Close()
					player1.Conn = nil
				}
				return
			}
			targetIdx := switchAction1.SwitchToIndex
			if targetIdx >= 0 && targetIdx < len(battleState.Player1Team) && !battleState.Player1Team[targetIdx].Fainted {
				battleState.Player1ActiveIndex = targetIdx
				turnSummary = append([]string{fmt.Sprintf("%s switched to %s!", player1.Username, battleState.Player1Team[targetIdx].Base.Name)}, turnSummary...)
				log.Printf("Turn %d: Player 1 switched to %s.", battleState.TurnNumber, battleState.Player1Team[targetIdx].Base.Name)
			} else {
				log.Printf("Turn %d: Player 1 %s sent invalid switch index %d. Ending game.", battleState.TurnNumber, player1.Username, targetIdx)
				turnSummary = append([]string{fmt.Sprintf("%s failed to choose a valid Pokemon!", player1.Username)}, turnSummary...)
				if player2.Conn != nil {
					server.SendResponse(player2.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player1.Username, "reason": "Invalid Switch Choice"}})
				}
				if player1.Conn != nil {
					player1.Conn.Close()
					player1.Conn = nil
				}
				return
			}
		}

		if p2FaintedThisTurn && !battle.IsAllFainted(battleState.Player2Team) {
			log.Printf("Turn %d: Player 2's %s fainted mid-turn. Requesting switch.", battleState.TurnNumber, p2FinalActing.Base.Name)
			server.SendResponse(player2.Conn, Response{Type: "switch_request", Message: map[string]interface{}{"reason": "Pokemon fainted"}})
			switchAction2, switchErr2 := receiveSwitchAction(player2.gameActionChan, player2.Username)
			if switchErr2 != nil {
				log.Printf("Turn %d: Error receiving switch action from %s: %v. Ending game.", battleState.TurnNumber, player2.Username, switchErr2)
				if player1.Conn != nil {
					server.SendResponse(player1.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player2.Username, "reason": "Switch Timeout/Error"}})
				}
				if player2.Conn != nil {
					player2.Conn.Close()
					player2.Conn = nil
				}
				return
			}
			targetIdx := switchAction2.SwitchToIndex
			if targetIdx >= 0 && targetIdx < len(battleState.Player2Team) && !battleState.Player2Team[targetIdx].Fainted {
				battleState.Player2ActiveIndex = targetIdx
				turnSummary = append([]string{fmt.Sprintf("%s switched to %s!", player2.Username, battleState.Player2Team[targetIdx].Base.Name)}, turnSummary...)
				log.Printf("Turn %d: Player 2 switched to %s.", battleState.TurnNumber, battleState.Player2Team[targetIdx].Base.Name)
			} else {
				log.Printf("Turn %d: Player 2 %s sent invalid switch index %d. Ending game.", battleState.TurnNumber, player2.Username, targetIdx)
				turnSummary = append([]string{fmt.Sprintf("%s failed to choose a valid Pokemon!", player2.Username)}, turnSummary...)
				if player1.Conn != nil {
					server.SendResponse(player1.Conn, Response{Type: "opponent_disconnected", Message: map[string]interface{}{"opponent": player2.Username, "reason": "Invalid Switch Choice"}})
				}
				if player2.Conn != nil {
					player2.Conn.Close()
					player2.Conn = nil
				}
				return
			}
		}

		log.Printf("Turn %d: Sending final results to %s and %s", battleState.TurnNumber, player1.Username, player2.Username)
		battleState.LastTurnResults = turnSummary
		p1SquadState := getSquadStateInfo(battleState.Player1Team)
		p2SquadState := getSquadStateInfo(battleState.Player2Team)
		resultMsgP1 := map[string]interface{}{
			"description":      turnSummary,
			"your_squad_state": p1SquadState, "opponent_squad_state": p2SquadState,
			"your_active_index": battleState.Player1ActiveIndex, "opponent_active_index": battleState.Player2ActiveIndex,
		}
		resultMsgP2 := map[string]interface{}{
			"description":      turnSummary,
			"your_squad_state": p2SquadState, "opponent_squad_state": p1SquadState,
			"your_active_index": battleState.Player2ActiveIndex, "opponent_active_index": battleState.Player1ActiveIndex,
		}
		if player1.Conn != nil {
			server.SendResponse(player1.Conn, Response{Type: "turn_result", Message: resultMsgP1})
		}
		if player2.Conn != nil {
			server.SendResponse(player2.Conn, Response{Type: "turn_result", Message: resultMsgP2})
		}

		p1LostAfterTurn := battle.IsAllFainted(battleState.Player1Team)
		p2LostAfterTurn := battle.IsAllFainted(battleState.Player2Team)
		if p1LostAfterTurn || p2LostAfterTurn {
			log.Printf("Game over detected *after* Turn %d results sent. P1 Lost: %v, P2 Lost: %v", battleState.TurnNumber, p1LostAfterTurn, p2LostAfterTurn)
			resultP1, resultP2 := "lose", "win"
			if p1LostAfterTurn && p2LostAfterTurn {
				resultP1, resultP2 = "draw", "draw"
			}
			if p1LostAfterTurn {
				resultP1, resultP2 = "lose", "win"
			}
			if p2LostAfterTurn {
				resultP1, resultP2 = "win", "lose"
			}
			if player1.Conn != nil {
				server.sendGameEnd(player1, player2, resultP1)
			}
			if player2.Conn != nil {
				server.sendGameEnd(player2, player1, resultP2)
			}
			return
		}

		battleState.TurnNumber++
	}
}

func receiveSwitchAction(actionChan <-chan []byte, username string) (PlayerAction, error) {
	var action PlayerAction
	receiveTimeout := time.After(65 * time.Second)

	select {
	case data, ok := <-actionChan:
		if !ok {
			return action, fmt.Errorf("action channel closed for %s during switch request", username)
		}
		msgStr := strings.TrimSpace(string(data))
		log.Printf("receiveSwitchAction (%s): Received data from channel: %q", username, msgStr)

		if strings.HasPrefix(msgStr, SwitchActionMarker+"|") {
			parts := strings.Split(msgStr, "|")
			if len(parts) == 2 {
				switchIdx, err := strconv.Atoi(parts[1])
				if err != nil {
					return action, fmt.Errorf("invalid number format in switch action from %s: %s", username, msgStr)
				}
				if switchIdx < 0 || switchIdx > 5 {
					return action, fmt.Errorf("invalid switch index %d from %s", switchIdx, username)
				}
				action.Type = "switch"
				action.SwitchToIndex = switchIdx
				log.Printf("Parsed switch action from %s: SwitchToIndex=%d", username, action.SwitchToIndex)
				return action, nil
			}
		}
		return action, fmt.Errorf("invalid switch action format received from %s: %s", username, msgStr)

	case <-receiveTimeout:
		return action, fmt.Errorf("timeout waiting for switch action from %s", username)
	}
}

func (server *Server) sendGameEnd(playerToSendTo, opponent *Client, result string) {
	if playerToSendTo == nil || playerToSendTo.Conn == nil {
		return
	}
	opponentUsername := "Opponent"
	if opponent != nil {
		opponentUsername = opponent.Username
	}
	message := fmt.Sprintf("Match against %s ended.", opponentUsername)
	switch result {
	case "win":
		message = fmt.Sprintf("You won the match against %s!", opponentUsername)
	case "lose":
		message = fmt.Sprintf("You lost the match against %s.", opponentUsername)
	case "draw":
		message = fmt.Sprintf("The match against %s ended in a draw!", opponentUsername)
	}
	log.Printf("Sending game_end to %s: Result=%s", playerToSendTo.Username, result)
	server.SendResponse(playerToSendTo.Conn, Response{
		Type:    "game_end",
		Message: map[string]interface{}{"result": result, "opponent": opponentUsername, "message": message},
	})
}

func extractMoveNames(moves []*pokemon.MoveInfo) []string {
	names := make([]string, 0, len(moves))
	if moves == nil {
		return names
	}
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

func receiveGameAction(actionChan <-chan []byte, username string, expectedMarker string) receivedAction {
	var action PlayerAction
	receiveTimeout := time.After(65 * time.Second)

	select {
	case data, ok := <-actionChan:
		if !ok {
			return receivedAction{err: fmt.Errorf("action channel closed for %s", username)}
		}
		msgStr := strings.TrimSpace(string(data))
		log.Printf("receiveGameAction (%s): Received data expecting '%s': %q", username, expectedMarker, msgStr)

		if strings.HasPrefix(msgStr, expectedMarker+"|") {
			prefixLen := len(expectedMarker) + 1
			payload := msgStr[prefixLen:]

			if expectedMarker == GameActionMarker {
				parts := strings.Split(payload, "|")
				if len(parts) == 3 {
					action.Type = parts[0]
					actionIdx, errIdx := strconv.Atoi(parts[1])
					switchIdx, errSwp := strconv.Atoi(parts[2])
					if errIdx != nil || errSwp != nil {
						return receivedAction{err: fmt.Errorf("invalid number format in game action from %s: %s", username, msgStr)}
					}
					if (action.Type != "move" && action.Type != "switch") || (action.Type == "move" && (actionIdx < 1 || actionIdx > 4)) || (action.Type == "switch" && (switchIdx < 0 || switchIdx > 5)) {
						return receivedAction{err: fmt.Errorf("invalid game action parameters from %s: %s", username, msgStr)}
					}
					action.ActionIndex = actionIdx
					action.SwitchToIndex = switchIdx
					log.Printf("Parsed game action from %s: Type=%s, ActionIndex=%d, SwitchToIndex=%d", username, action.Type, action.ActionIndex, action.SwitchToIndex)
					return receivedAction{action: action, err: nil}
				}
			} else if expectedMarker == SwitchActionMarker {
				switchIdx, err := strconv.Atoi(payload)
				if err != nil {
					return receivedAction{err: fmt.Errorf("invalid number format in switch action from %s: %s", username, msgStr)}
				}
				if switchIdx < 0 || switchIdx > 5 {
					return receivedAction{err: fmt.Errorf("invalid switch index %d from %s", switchIdx, username)}
				}
				action.Type = "switch"
				action.SwitchToIndex = switchIdx
				log.Printf("Parsed switch action from %s: SwitchToIndex=%d", username, action.SwitchToIndex)
				return receivedAction{action: action, err: nil}
			}
		}
		return receivedAction{err: fmt.Errorf("invalid/unexpected action format received from %s: %s", username, msgStr)}

	case <-receiveTimeout:
		return receivedAction{err: fmt.Errorf("timeout waiting for action (%s) from %s", expectedMarker, username)}
	}
}

func getMoveFromAction(action PlayerAction, moves []*pokemon.MoveInfo) *pokemon.MoveInfo {
	if action.Type != "move" {
		return nil
	}
	if moves == nil {
		log.Printf("Error: Attempted to get move from nil moveset (Action: %+v)", action)
		return nil
	}
	if action.ActionIndex < 1 || action.ActionIndex > len(moves) {
		log.Printf("Invalid move index received in getMoveFromAction: %d (available: %d)", action.ActionIndex, len(moves))
		return nil
	}
	move := moves[action.ActionIndex-1]
	if move == nil {
		log.Printf("Warning: Move at index %d is nil. Action: %+v", action.ActionIndex-1, action)
	}
	return move
}
