package client

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func (c *Client) processPlayerList(msg Message) {
	playersInterface, ok := msg.Message["players"]
	if !ok {
		log.Println("No players field in player_list message")
		return
	}
	players, ok := playersInterface.([]interface{})
	if !ok {
		log.Printf("Invalid players format: %T", playersInterface)
		return
	}

	fmt.Println("\nConnected players:")
	hasPlayers := false
	for _, player := range players {
		playerName, ok := player.(string)
		if !ok {
			continue
		}
		hasPlayers = true
		fmt.Printf("- %s", playerName)
		if playerName == c.Config.Username {
			fmt.Print(" (you)")
		}
		fmt.Println()
	}
	if !hasPlayers {
		fmt.Println("(No other players online)")
	}
	fmt.Print("> ")
}

func (c *Client) processMatchStart(msg Message) {
	opponentInterface, ok := msg.Message["opponent"]
	if !ok {
		log.Println("No opponent field in match_start message")
		return
	}
	opponent, ok := opponentInterface.(string)
	if !ok {
		log.Printf("Invalid opponent format: %T", opponentInterface)
		return
	}

	c.Opponent = opponent
	c.InMatch = true
	fmt.Printf("\nMatch found with %s! Waiting for game to start...\n", opponent)
}

func (c *Client) processGameStart(msg Message) {
	yourSquadInterface, ok := msg.Message["your_squad"]
	if !ok {
		log.Println("No your_squad field in game_start message")
		return
	}
	opponentSquadInterface, ok := msg.Message["opponent_squad"]
	if !ok {
		log.Println("No opponent_squad field in game_start message")
		return
	}
	yourSquadRaw, ok := yourSquadInterface.([]interface{})
	if !ok {
		log.Printf("Invalid your_squad format: %T", yourSquadInterface)
		return
	}
	opponentSquadRaw, ok := opponentSquadInterface.([]interface{})
	if !ok {
		log.Printf("Invalid opponent_squad format: %T", opponentSquadInterface)
		return
	}

	yourSquad := make([]string, len(yourSquadRaw))
	for i, v := range yourSquadRaw {
		yourSquad[i], _ = v.(string)
	}
	opponentSquad := make([]string, len(opponentSquadRaw))
	for i, v := range opponentSquadRaw {
		opponentSquad[i], _ = v.(string)
	}

	fmt.Printf("\n=== Battle Start vs %s ===\n", c.Opponent)
	fmt.Println("\nYour squad:")
	for _, pokemon := range yourSquad {
		fmt.Printf("- %s\n", pokemon)
	}
	fmt.Println("\nOpponent's squad:")
	for _, pokemon := range opponentSquad {
		fmt.Printf("- %s\n", pokemon)
	}

	c.setupBattleState(yourSquad, opponentSquad)

	c.startGameMode()
}

func (c *Client) handleSwitchRequest(msg Message) {
	log.Println("Received switch request from server.")
	reason, _ := msg.Message["reason"].(string)

	if !c.GameActive {
		log.Println("Warning: Received switch_request while not in game mode.")
		return
	}

	fmt.Printf("\n--- Forced Switch Required! (%s) ---\n", reason)

	fmt.Println("Available Pokemon:")
	availableSwitchIndices := []int{}
	if c.PlayerSquad != nil {
		for i, poke := range c.PlayerSquad {
			if poke != nil && !poke.Fainted {
				maxHP := 0.0
				if i < len(c.PlayerMaxHPs) {
					maxHP = c.PlayerMaxHPs[i]
				}
				hpPercent := 0.0
				if maxHP > 0 {
					hpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
				}
				fmt.Printf("%d. %-12s HP: %3.0f%%\n", i+1, poke.Base.Name, hpPercent)
				availableSwitchIndices = append(availableSwitchIndices, i)
			}
		}
	}

	if len(availableSwitchIndices) == 0 {
		fmt.Println("!!! No available Pokemon to switch to! Notifying server (Error).")
		log.Println("Error: No valid Pokemon to switch to during forced switch.")
		c.sendSwitchAction(-1)
		return
	}

	c.AwaitingForcedSwitch = true

	fmt.Print("Enter the number of the Pokemon to switch to: ")
}

func (c *Client) processGameEnd(msg Message) {
	resultInterface, ok := msg.Message["result"]
	if !ok {
		log.Println("No result field in game_end message")
		return
	}
	result, ok := resultInterface.(string)
	if !ok {
		log.Printf("Invalid result format: %T", resultInterface)
		return
	}

	if c.GameActive || c.AwaitingForcedSwitch {
		c.endGameMode()
	}

	opponent := c.Opponent
	c.InMatch = false
	c.Opponent = ""

	fmt.Println("\n=== Battle End ===")
	switch result {
	case "win":
		fmt.Printf("You won the match against %s!\n", opponent)
	case "lose":
		fmt.Printf("You lost the match against %s.\n", opponent)
	case "draw":
		fmt.Printf("The match against %s ended in a draw.\n", opponent)
	default:
		fmt.Printf("Match with %s ended: %s\n", opponent, result)
	}
	fmt.Print("> ")
}

func (c *Client) applyBattleStateUpdate(msg Message) {
	log.Println("Applying battle state update...")
	var yourSquadUpdate []PokemonStateInfo
	var opponentSquadUpdate []PokemonStateInfo

	if data, ok := msg.Message["your_squad_state"]; ok {
		jsonBytes, err := json.Marshal(data)
		if err == nil {
			err = json.Unmarshal(jsonBytes, &yourSquadUpdate)
		}
		if err != nil {
			log.Printf("Error un/marshaling your_squad_state: %v", err)
		}
	} else {
		log.Println("Warning: 'your_squad_state' missing from update message")
	}

	if data, ok := msg.Message["opponent_squad_state"]; ok {
		jsonBytes, err := json.Marshal(data)
		if err == nil {
			err = json.Unmarshal(jsonBytes, &opponentSquadUpdate)
		}
		if err != nil {
			log.Printf("Error un/marshaling opponent_squad_state: %v", err)
		}
	} else {
		log.Println("Warning: 'opponent_squad_state' missing from update message")
	}

	if c.PlayerSquad != nil && len(yourSquadUpdate) > 0 {
		if len(yourSquadUpdate) != len(c.PlayerSquad) {
			log.Printf("Warning: Player squad update length mismatch. Local=%d, Update=%d", len(c.PlayerSquad), len(yourSquadUpdate))
		} else {
			for i, updateInfo := range yourSquadUpdate {
				if i < len(c.PlayerSquad) && c.PlayerSquad[i] != nil && c.PlayerSquad[i].Base != nil {
					if c.PlayerSquad[i].Base.Name == updateInfo.Name {
						c.PlayerSquad[i].CurrentHP = updateInfo.CurrentHP
						c.PlayerSquad[i].Fainted = updateInfo.Fainted
						c.PlayerSquad[i].Status = updateInfo.Status
					} else {
						log.Printf("Warning: Name mismatch at index %d during player state update. Expected %s, got %s", i, c.PlayerSquad[i].Base.Name, updateInfo.Name)
					}
				} else {
					log.Printf("Warning: Nil BattlePokemon/Base or index out of bounds (%d) in PlayerSquad during update.", i)
				}
			}
		}
	} else {
		log.Printf("Warning: Player squad update skipped. Local squad nil or update empty.")
	}

	if c.EnemySquad != nil && len(opponentSquadUpdate) > 0 {
		if len(opponentSquadUpdate) != len(c.EnemySquad) {
			log.Printf("Warning: Opponent squad update length mismatch. Local=%d, Update=%d", len(c.EnemySquad), len(opponentSquadUpdate))
		} else {
			for i, updateInfo := range opponentSquadUpdate {
				if i < len(c.EnemySquad) && c.EnemySquad[i] != nil && c.EnemySquad[i].Base != nil {
					if c.EnemySquad[i].Base.Name == updateInfo.Name {
						c.EnemySquad[i].CurrentHP = updateInfo.CurrentHP
						c.EnemySquad[i].Fainted = updateInfo.Fainted
						c.EnemySquad[i].Status = updateInfo.Status
					} else {
						log.Printf("Warning: Name mismatch at index %d during opponent state update. Expected %s, got %s", i, c.EnemySquad[i].Base.Name, updateInfo.Name)
					}
				} else {
					log.Printf("Warning: Nil BattlePokemon/Base or index out of bounds (%d) in EnemySquad during update.", i)
				}
			}
		}
	} else {
		log.Printf("Warning: Opponent squad update skipped. Local squad nil or update empty.")
	}

	if idxFloat, ok := msg.Message["your_active_index"].(float64); ok {
		c.PlayerActiveIdx = int(idxFloat)
	} else {
		log.Println("Warning: 'your_active_index' missing or invalid in update message")
	}
	if idxFloat, ok := msg.Message["opponent_active_index"].(float64); ok {
		c.EnemyActiveIdx = int(idxFloat)
	} else {
		log.Println("Warning: 'opponent_active_index' missing or invalid in update message")
	}

	if descInterface, ok := msg.Message["description"].([]interface{}); ok {
		c.LastTurnDescription = make([]string, len(descInterface))
		for i, desc := range descInterface {
			if descStr, ok := desc.(string); ok {
				c.LastTurnDescription[i] = descStr
			}
		}
	} else {
		c.LastTurnDescription = []string{"(No description received)"}
	}

	log.Println("Battle state update applied.")
}

func (c *Client) setupBattleState(yourSquadNames, opponentSquadNames []string) {
	log.Println("Setting up client battle state by fetching data...")
	startTime := time.Now()
	moveCache := make(map[string]*pokemon.MoveInfo)
	var cacheMutex sync.Mutex
	fetchMoveWithCache := func(url string) (*pokemon.MoveInfo, error) {
		cacheMutex.Lock()
		if cachedMove, found := moveCache[url]; found {
			cacheMutex.Unlock()
			return cachedMove, nil
		}
		cacheMutex.Unlock()
		var moveData *pokemon.MoveInfo
		var err error
		maxRetries := 3
		for attempts := 0; attempts < maxRetries; attempts++ {
			moveData, err = pokemon.FetchMoveData(url)
			if err == nil {
				cacheMutex.Lock()
				moveCache[url] = moveData
				cacheMutex.Unlock()
				return moveData, nil
			}
			log.Printf("Attempt %d: Error fetching move %s: %v. Retrying...", attempts+1, url, err)
			time.Sleep(time.Duration(100*(attempts+1)) * time.Millisecond)
		}
		return nil, fmt.Errorf("failed to fetch move %s after %d retries: %w", url, maxRetries, err)
	}

	var wg sync.WaitGroup
	var setupMutex sync.Mutex
	playerSquadSize := len(yourSquadNames)
	enemySquadSize := len(opponentSquadNames)
	c.PlayerSquad = make([]*battle.BattlePokemon, playerSquadSize)
	c.EnemySquad = make([]*battle.BattlePokemon, enemySquadSize)
	c.PlayerMaxHPs = make([]float64, playerSquadSize)
	c.EnemyMaxHPs = make([]float64, enemySquadSize)

	processPokemon := func(idx int, pokeName string, isPlayer bool) {
		defer wg.Done()
		log.Printf("Initializing %s (%s)...", pokeName, map[bool]string{true: "Player", false: "Opponent"}[isPlayer])
		basePoke, err := pokemon.FetchPokemonData(pokeName)
		if err != nil || basePoke == nil {
			log.Printf("Error fetching base data for %s: %v", pokeName, err)
			setupMutex.Lock()
			if isPlayer && idx < len(c.PlayerSquad) {
				c.PlayerSquad[idx] = nil
			}
			if !isPlayer && idx < len(c.EnemySquad) {
				c.EnemySquad[idx] = nil
			}
			setupMutex.Unlock()
			return
		}
		moveEntries := pokemon.PickRandMoves(basePoke)
		moveset := []*pokemon.MoveInfo{}
		var moveWg sync.WaitGroup
		var movesetMutex sync.Mutex
		for _, entry := range moveEntries {
			moveWg.Add(1)
			go func(mEntry pokemon.ApiResource) {
				defer moveWg.Done()
				moveData, err := fetchMoveWithCache(mEntry.URL)
				if err != nil {
					log.Printf("Error fetching move details %s for %s: %v", mEntry.Name, pokeName, err)
					return
				}
				if moveData != nil {
					movesetMutex.Lock()
					moveset = append(moveset, moveData)
					movesetMutex.Unlock()
				}
			}(entry)
		}
		moveWg.Wait()
		battlePoke := battle.NewBattlePokemon(basePoke, moveset)
		if battlePoke == nil {
			log.Printf("Error creating BattlePokemon for %s", pokeName)
			setupMutex.Lock()
			setupMutex.Unlock()
			return
		}
		maxHP := 0.0
		hpBaseStat := stats.GetStat(battlePoke.Base, "hp")
		if hpBaseStat > 0 {
			maxHP = stats.HpCalc(hpBaseStat)
		} else {
			log.Printf("Warning: Could not get HP base stat for %s", battlePoke.Base.Name)
		}
		battlePoke.CurrentHP = maxHP
		setupMutex.Lock()
		if isPlayer {
			if idx < len(c.PlayerSquad) {
				c.PlayerSquad[idx] = battlePoke
				c.PlayerMaxHPs[idx] = maxHP
			}
		} else {
			if idx < len(c.EnemySquad) {
				c.EnemySquad[idx] = battlePoke
				c.EnemyMaxHPs[idx] = maxHP
			}
		}
		setupMutex.Unlock()
		log.Printf("Initialized %s (MaxHP: %.1f)", pokeName, maxHP)
	}

	log.Println("Initializing Player Squad...")
	for i, name := range yourSquadNames {
		wg.Add(1)
		go processPokemon(i, name, true)
	}
	log.Println("Initializing Opponent Squad...")
	for i, name := range opponentSquadNames {
		wg.Add(1)
		go processPokemon(i, name, false)
	}
	wg.Wait()
	c.PlayerActiveIdx = 0
	c.EnemyActiveIdx = 0
	squadPopulated := true
	setupMutex.Lock()
	for i := range c.PlayerSquad {
		if c.PlayerSquad[i] == nil {
			squadPopulated = false
			log.Printf("Error: Player squad member at index %d failed.", i)
		}
	}
	for i := range c.EnemySquad {
		if c.EnemySquad[i] == nil {
			squadPopulated = false
			log.Printf("Error: Opponent squad member at index %d failed.", i)
		}
	}
	setupMutex.Unlock()
	if !squadPopulated {
		log.Println("Error: Failed to initialize one or more Pokemon.")
	}
	log.Printf("Client battle state setup complete. Time: %s", time.Since(startTime))
	fmt.Println("\nBattle state ready!")
}

func (c *Client) sendSwitchAction(switchIndex int) {
	actionStr := fmt.Sprintf("%s|%d", SwitchActionMarker, switchIndex)
	log.Printf("Sending forced switch action: %s", actionStr)

	if c.Conn == nil || !c.Connected {
		log.Println("Error: Cannot send switch action, not connected.")
		fmt.Println("Error: Connection lost. Cannot send switch action.")
		c.Disconnect()
		return
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Conn.Write([]byte(actionStr))
	c.Conn.SetWriteDeadline(time.Time{})

	if err != nil {
		log.Printf("Failed to send switch action: %v", err)
		fmt.Println("Error sending switch action to server. Disconnecting.")
		c.Disconnect()
	} else {
		fmt.Println("Sent switch action to server...")
		c.AwaitingForcedSwitch = false
	}
}
