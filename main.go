package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func main() {
	start := time.Now()

	// Setup full squads instead of single Pokémon
	playerSquad, enemySquad, playerMovesets, enemyMovesets := setupFullSquads()

	// Track current active Pokémon
	playerActiveIndex := 0
	enemyActiveIndex := 0

	// Get current active Pokémon
	playerBattlePokemon := playerSquad[playerActiveIndex]
	enemyBattlePokemon := enemySquad[enemyActiveIndex]

	// Track max HP for each Pokémon for percentage calculations
	playerMaxHPs := make([]float64, len(playerSquad))
	enemyMaxHPs := make([]float64, len(enemySquad))

	for i, p := range playerSquad {
		playerMaxHPs[i] = p.CurrentHP
	}

	for i, p := range enemySquad {
		enemyMaxHPs[i] = p.CurrentHP
	}

	for {
		// Display battle status
		displayBattleStatus(playerBattlePokemon, enemyBattlePokemon, playerMaxHPs[playerActiveIndex], enemyMaxHPs[enemyActiveIndex])

		// Show action menu
		fmt.Println("\nWhat would you like to do?")
		fmt.Println("1. Fight")
		fmt.Println("2. Switch Pokémon")

		var choice int
		fmt.Print("Enter your choice (1-2): ")
		fmt.Scan(&choice)

		if choice == 1 {
			// FIGHT OPTION
			// Show the moveset and PP for each move
			fmt.Println("\nYour moveset and remaining PP:")
			for i, move := range playerMovesets[playerActiveIndex] {
				fmt.Printf("%d. %s (PP: %d)\n", i+1, move.Name, playerBattlePokemon.MovePP[move.Name])
			}

			// User selects a move
			var selectedMove int
			fmt.Print("\nSelect your move (enter a number 1 - 4): ")
			fmt.Scan(&selectedMove)

			// Validate move selection
			if selectedMove < 1 || selectedMove > len(playerMovesets[playerActiveIndex]) {
				fmt.Println("Invalid move selection. Please try again.")
				continue
			}

			// Get the move data
			moveData := playerMovesets[playerActiveIndex][selectedMove-1]

			// Check PP
			if playerBattlePokemon.MovePP[moveData.Name] == 0 {
				fmt.Println("This move has no PP left. Please select another move.")
				continue
			}

			// Use the move
			if !playerBattlePokemon.UseMove(moveData.Name) {
				fmt.Println("Move failed or has no PP left.")
				continue
			}

			// Display move information
			fmt.Printf("You used: %s, Move accuracy: %d, Move Power: %d\n",
				moveData.Name, moveData.Accuracy, moveData.Power)

			// Calculate and apply damage
			damage, percent := battle.DamageCalc(playerBattlePokemon.Base, enemyBattlePokemon.Base, moveData)
			enemyBattlePokemon.ApplyDamage(damage)

			fmt.Printf("You dealt %d damage! (~%.2f%% of %s's HP)\n",
				damage, percent, enemyBattlePokemon.Base.Name)

		} else if choice == 2 {
			// SWITCH OPTION
			fmt.Println("\nYour Pokémon squad:")
			for i, poke := range playerSquad {
				status := "Ready"
				if poke.Fainted {
					status = "Fainted"
				} else if i == playerActiveIndex {
					status = "Active"
				}

				if poke.Status != "" {
					status += " (" + poke.Status + ")"
				}

				hpPercent := (poke.CurrentHP / playerMaxHPs[i]) * 100
				fmt.Printf("%d. %s - HP: %.2f/%.2f (%.2f%%) - Status: %s\n",
					i+1, poke.Base.Name, poke.CurrentHP, playerMaxHPs[i], hpPercent, status)
			}

			var newIndex int
			for {
				fmt.Print("\nSelect a Pokémon to switch to (enter a number 1 - 6): ")
				fmt.Scan(&newIndex)
				newIndex-- // Convert to 0-based index

				if newIndex < 0 || newIndex >= len(playerSquad) {
					fmt.Println("Invalid selection. Please try again.")
					continue
				}

				if newIndex == playerActiveIndex {
					fmt.Println("This Pokémon is already active.")
					continue
				}

				if playerSquad[newIndex].Fainted {
					fmt.Println("This Pokémon has fainted and cannot battle.")
					continue
				}

				break
			}

			// Switch Pokémon
			playerActiveIndex = newIndex
			playerBattlePokemon = playerSquad[playerActiveIndex]
			fmt.Printf("You switched to %s!\n", playerBattlePokemon.Base.Name)
		} else {
			fmt.Println("Invalid choice. Please try again.")
			continue
		}

		// Check if enemy fainted
		if enemyBattlePokemon.Fainted {
			// Find a non-fainted Pokémon
			newEnemyIndex := -1
			for i, poke := range enemySquad {
				if !poke.Fainted && i != enemyActiveIndex {
					newEnemyIndex = i
					break
				}
			}

			if newEnemyIndex == -1 {
				fmt.Println("\nAll enemy Pokémon have fainted! You win!")
				break
			}

			fmt.Printf("\nEnemy's %s has fainted!\n", enemyBattlePokemon.Base.Name)
			enemyActiveIndex = newEnemyIndex
			enemyBattlePokemon = enemySquad[enemyActiveIndex]
			fmt.Printf("Enemy sent out %s!\n", enemyBattlePokemon.Base.Name)
			continue // Skip enemy's turn since switching takes a turn
		}

		// Enemy's turn - randomly decide to attack or switch (20% chance to switch)
		if rand.Float64() < 0.2 {
			// Try to switch
			availablePokemon := []int{}
			for i, poke := range enemySquad {
				if !poke.Fainted && i != enemyActiveIndex {
					availablePokemon = append(availablePokemon, i)
				}
			}

			if len(availablePokemon) > 0 {
				newEnemyIndex := availablePokemon[rand.Intn(len(availablePokemon))]
				enemyActiveIndex = newEnemyIndex
				enemyBattlePokemon = enemySquad[enemyActiveIndex]
				fmt.Printf("Enemy switched to %s!\n", enemyBattlePokemon.Base.Name)
			} else {
				// No available Pokémon to switch to, attack instead
				enemyAttack(enemyBattlePokemon, playerBattlePokemon, enemyMovesets[enemyActiveIndex])
			}
		} else {
			// Attack
			enemyAttack(enemyBattlePokemon, playerBattlePokemon, enemyMovesets[enemyActiveIndex])
		}

		// Check if player's Pokémon fainted
		if playerBattlePokemon.Fainted {
			// Check if any Pokémon can still battle
			allFainted := true
			for _, poke := range playerSquad {
				if !poke.Fainted {
					allFainted = false
					break
				}
			}

			if allFainted {
				fmt.Println("\nAll your Pokémon have fainted! You lose!")
				break
			}

			fmt.Printf("\nYour %s has fainted! You must switch to another Pokémon.\n",
				playerBattlePokemon.Base.Name)

			// Select a new Pokémon
			fmt.Println("\nYour remaining Pokémon:")
			for i, poke := range playerSquad {
				if !poke.Fainted {
					hpPercent := (poke.CurrentHP / playerMaxHPs[i]) * 100
					fmt.Printf("%d. %s - HP: %.2f/%.2f (%.2f%%)\n",
						i+1, poke.Base.Name, poke.CurrentHP, playerMaxHPs[i], hpPercent)
				}
			}

			var newIndex int
			for {
				fmt.Print("\nSelect a Pokémon to send out (enter a number 1 - 6): ")
				fmt.Scan(&newIndex)
				newIndex-- // Convert to 0-based index

				if newIndex < 0 || newIndex >= len(playerSquad) {
					fmt.Println("Invalid selection. Please try again.")
					continue
				}

				if playerSquad[newIndex].Fainted {
					fmt.Println("This Pokémon has fainted and cannot battle.")
					continue
				}

				break
			}

			playerActiveIndex = newIndex
			playerBattlePokemon = playerSquad[playerActiveIndex]
			fmt.Printf("You sent out %s!\n", playerBattlePokemon.Base.Name)
		}

		time.Sleep(1 * time.Second)
	}

	elapsed := time.Since(start)
	fmt.Println("\nExecution Time:", elapsed)
}

// Helper function for enemy attacks
func enemyAttack(attacker, defender *battle.BattlePokemon, moveSet []*pokemon.MoveInfo) {
	opponentMoveIndex := rand.Intn(len(moveSet))
	opponentMoveData := moveSet[opponentMoveIndex]

	if attacker.MovePP[opponentMoveData.Name] > 0 {
		attacker.UseMove(opponentMoveData.Name)
		opponentDamage, opponentPercent := battle.DamageCalc(attacker.Base, defender.Base, opponentMoveData)
		defender.ApplyDamage(opponentDamage)

		fmt.Printf("%s used %s! It dealt %d damage! (~%.2f%% of your Pokémon's HP)\n",
			attacker.Base.Name, opponentMoveData.Name, opponentDamage, opponentPercent)
	} else {
		fmt.Printf("%s tried to use %s but has no PP left!\n",
			attacker.Base.Name, opponentMoveData.Name)
	}
}

// Display battle status helper
func displayBattleStatus(player, enemy *battle.BattlePokemon, playerMaxHP, enemyMaxHP float64) {
	playerHPPercent := (player.CurrentHP / playerMaxHP) * 100
	enemyHPPercent := (enemy.CurrentHP / enemyMaxHP) * 100

	fmt.Printf("\nYour %s's HP: %.2f/%.2f (%.2f%%)",
		player.Base.Name, player.CurrentHP, playerMaxHP, playerHPPercent)
	if player.Status != "" {
		fmt.Printf(" [%s]", player.Status)
	}
	fmt.Println()

	fmt.Printf("Enemy %s's HP: %.2f/%.2f (%.2f%%)",
		enemy.Base.Name, enemy.CurrentHP, enemyMaxHP, enemyHPPercent)
	if enemy.Status != "" {
		fmt.Printf(" [%s]", enemy.Status)
	}
	fmt.Println()
}

// Function to setup full squads
func setupFullSquads() ([]*battle.BattlePokemon, []*battle.BattlePokemon, [][]*pokemon.MoveInfo, [][]*pokemon.MoveInfo) {
	totalStartTime := time.Now()

	// Create a cache to store move data
	moveCache := make(map[string]*pokemon.MoveInfo)
	var cacheMutex sync.Mutex

	// Helper function to fetch move data with caching and retries
	fetchMoveWithCache := func(url string) (*pokemon.MoveInfo, error) {
		// Check cache first
		cacheMutex.Lock()
		if cachedMove, found := moveCache[url]; found {
			cacheMutex.Unlock()
			return cachedMove, nil
		}
		cacheMutex.Unlock()

		// Not in cache, fetch it (with retries)
		var moveData *pokemon.MoveInfo
		var err error
		maxRetries := 3

		for attempts := 0; attempts < maxRetries; attempts++ {
			// Custom implementation to use our HTTP client
			// Note: You'll need to modify FetchMoveData to accept a client parameter
			// or implement the fetch logic directly here
			moveData, err = pokemon.FetchMoveData(url)

			if err == nil {
				// Success - add to cache and return
				cacheMutex.Lock()
				moveCache[url] = moveData
				cacheMutex.Unlock()
				return moveData, nil
			}

			// Failed - wait briefly before retry
			time.Sleep(200 * time.Millisecond)
		}

		return nil, err
	}

	playerSquadBase := pokemon.SelectRandSquad()
	enemySquadBase := pokemon.SelectRandSquad()

	fmt.Println("Your randomly selected pokemon squad is:")
	for i := range playerSquadBase {
		fmt.Println(i+1, playerSquadBase[i].Name)
	}

	fmt.Println("\nEnemy randomly selected pokemon squad is:")
	for i := range enemySquadBase {
		fmt.Println(i+1, enemySquadBase[i].Name)
	}

	var playerSelect int
	fmt.Print("\nSelect your first Pokémon to send out (enter a number 1 - 6): ")
	fmt.Scan(&playerSelect)
	playerSelect = (playerSelect - 1) % len(playerSquadBase)

	enemySelect := rand.Intn(len(enemySquadBase))

	fmt.Printf("You sent out %s!\n", playerSquadBase[playerSelect].Name)
	fmt.Printf("Enemy sent out %s!\n", enemySquadBase[enemySelect].Name)

	// Initialize full squads with movesets
	playerSquad := make([]*battle.BattlePokemon, len(playerSquadBase))
	enemySquad := make([]*battle.BattlePokemon, len(enemySquadBase))

	playerMovesets := make([][]*pokemon.MoveInfo, len(playerSquadBase))
	enemyMovesets := make([][]*pokemon.MoveInfo, len(enemySquadBase))

	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Println("\nLoading movesets in parallel (with optimizations)...")

	// Setup player squad with optimized goroutines
	for i, base := range playerSquadBase {
		wg.Add(1)
		go func(i int, base *pokemon.Pokemon) {
			defer wg.Done()

			pokeStartTime := time.Now()

			mu.Lock()
			fmt.Printf("Fetching moveset for your %s...\n", base.Name)
			mu.Unlock()

			moves := pokemon.PickRandMoves(base)
			moveset := []*pokemon.MoveInfo{}

			// Create a wait group for the moves of this Pokémon
			var moveWg sync.WaitGroup
			var movesetMutex sync.Mutex

			// Fetch moves in parallel too
			for _, moveAPI := range moves {
				moveWg.Add(1)
				go func(moveURL string) {
					defer moveWg.Done()

					moveData, err := fetchMoveWithCache(moveURL)
					if err != nil {
						mu.Lock()
						fmt.Printf("Error fetching move for %s: %v\n", base.Name, err)
						mu.Unlock()
						return
					}

					movesetMutex.Lock()
					moveset = append(moveset, moveData)
					movesetMutex.Unlock()
				}(moveAPI.URL)
			}

			// Wait for all moves to be fetched
			moveWg.Wait()

			playerMovesets[i] = moveset
			playerSquad[i] = battle.NewBattlePokemon(base, moveset)

			loadTime := time.Since(pokeStartTime)

			mu.Lock()
			fmt.Printf("Completed loading moveset for your %s! (%.2f seconds)\n",
				base.Name, loadTime.Seconds())
			mu.Unlock()
		}(i, base)
	}

	// Setup enemy squad with optimized goroutines (similar approach)
	for i, base := range enemySquadBase {
		wg.Add(1)
		go func(i int, base *pokemon.Pokemon) {
			defer wg.Done()

			pokeStartTime := time.Now()

			mu.Lock()
			fmt.Printf("Fetching moveset for enemy %s...\n", base.Name)
			mu.Unlock()

			moves := pokemon.PickRandMoves(base)
			moveset := []*pokemon.MoveInfo{}

			// Create a wait group for the moves of this Pokémon
			var moveWg sync.WaitGroup
			var movesetMutex sync.Mutex

			// Fetch moves in parallel too
			for _, moveAPI := range moves {
				moveWg.Add(1)
				go func(moveURL string) {
					defer moveWg.Done()

					moveData, err := fetchMoveWithCache(moveURL)
					if err != nil {
						mu.Lock()
						fmt.Printf("Error fetching move for enemy %s: %v\n", base.Name, err)
						mu.Unlock()
						return
					}

					movesetMutex.Lock()
					moveset = append(moveset, moveData)
					movesetMutex.Unlock()
				}(moveAPI.URL)
			}

			// Wait for all moves to be fetched
			moveWg.Wait()

			enemyMovesets[i] = moveset
			enemySquad[i] = battle.NewBattlePokemon(base, moveset)

			loadTime := time.Since(pokeStartTime)

			mu.Lock()
			fmt.Printf("Completed loading moveset for enemy %s! (%.2f seconds)\n",
				base.Name, loadTime.Seconds())
			mu.Unlock()
		}(i, base)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	totalLoadTime := time.Since(totalStartTime)
	fmt.Printf("\nAll movesets loaded successfully! Total loading time: %.2f seconds\n",
		totalLoadTime.Seconds())

	return playerSquad, enemySquad, playerMovesets, enemyMovesets
}

