package client

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
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

	fmt.Println("Connected players:")
	for i, player := range players {
		playerName, ok := player.(string)
		if !ok {
			continue
		}
		fmt.Printf("%d. %s", i+1, playerName)
		if playerName == c.Config.Username {
			fmt.Print(" (you)")
		}
		fmt.Println()
	}
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
	fmt.Printf("Match started with %s!\n", opponent)
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
		yourSquad[i], ok = v.(string)
		if !ok {
			log.Printf("Non-string pokemon in your_squad: %T", v)
			return
		}
	}

	opponentSquad := make([]string, len(opponentSquadRaw))
	for i, v := range opponentSquadRaw {
		opponentSquad[i], ok = v.(string)
		if !ok {
			log.Printf("Non-string pokemon in opponent_squad: %T", v)
			return
		}
	}

	c.InMatch = true

	fmt.Printf("Match started against %s!\n", c.Opponent)
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

	if c.GameActive {
		c.endGameMode()
	}

	c.InMatch = false
	opponent := c.Opponent
	c.Opponent = ""

	switch result {
	case "win":
		fmt.Printf("You won the match against %s!\n", opponent)
	case "lose":
		fmt.Printf("You lost the match against %s.\n", opponent)
	default:
		fmt.Printf("Match with %s ended: %s\n", opponent, result)
	}
}

func (c *Client) processGameUpdate(msg Message) {

	if messages, ok := msg.Message["battle_messages"].([]interface{}); ok {
		for _, msgInterface := range messages {
			if battleMsg, ok := msgInterface.(string); ok {
				fmt.Println(battleMsg)
			}
		}
	}
}

// Add this method to set up the battle state
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

	processPokemon := func(idx int, pokeName string, isPlayer bool) {
		defer wg.Done()
		log.Printf("Initializing %s (%s)...", pokeName, map[bool]string{true: "Player", false: "Opponent"}[isPlayer])

		basePoke, err := pokemon.FetchPokemonData(pokeName)
		if err != nil {
			log.Printf("Error fetching base data for %s: %v", pokeName, err)
			return
		}
		if basePoke == nil {
			log.Printf("Error: Fetched nil base data for %s", pokeName)
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
			return
		}

		setupMutex.Lock()
		if isPlayer {
			if idx < len(c.PlayerSquad) {
				c.PlayerSquad[idx] = battlePoke
			}
		} else {
			if idx < len(c.EnemySquad) {
				c.EnemySquad[idx] = battlePoke
			}
		}
		setupMutex.Unlock()
		log.Printf("Initialized %s", pokeName)
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
	for i := range c.PlayerSquad {
		if c.PlayerSquad[i] == nil {
			squadPopulated = false
			log.Printf("Error: Player squad member at index %d failed to initialize.", i)
		}
	}
	for i := range c.EnemySquad {
		if c.EnemySquad[i] == nil {
			squadPopulated = false
			log.Printf("Error: Opponent squad member at index %d failed to initialize.", i)
		}
	}

	if !squadPopulated {
		log.Println("Error: Failed to initialize one or more Pokemon in the squads.")
	}

	log.Printf("Client battle state setup complete. Time: %s", time.Since(startTime))
	fmt.Println("\nBattle state ready!")
}
