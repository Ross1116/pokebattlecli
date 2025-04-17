package battle

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func SetupFullSquads() ([]*BattlePokemon, []*BattlePokemon, [][]*pokemon.MoveInfo, [][]*pokemon.MoveInfo) {
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
	playerSquad := make([]*BattlePokemon, len(playerSquadBase))
	enemySquad := make([]*BattlePokemon, len(enemySquadBase))

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
			playerSquad[i] = NewBattlePokemon(base, moveset)

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
			enemySquad[i] = NewBattlePokemon(base, moveset)

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
