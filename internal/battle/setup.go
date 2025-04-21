package battle

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func SetupFullSquads() ([]*BattlePokemon, []*BattlePokemon, [][]*pokemon.MoveInfo, [][]*pokemon.MoveInfo, int, int) {
	totalStartTime := time.Now()

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
	playerActiveIndex := playerSelect

	enemySelect := rand.Intn(len(enemySquadBase))
	enemyActiveIndex := enemySelect

	fmt.Printf("You sent out %s!\n", playerSquadBase[playerSelect].Name)
	fmt.Printf("Enemy sent out %s!\n", enemySquadBase[enemySelect].Name)

	playerSquad := make([]*BattlePokemon, len(playerSquadBase))
	enemySquad := make([]*BattlePokemon, len(enemySquadBase))

	playerMovesets := make([][]*pokemon.MoveInfo, len(playerSquadBase))
	enemyMovesets := make([][]*pokemon.MoveInfo, len(enemySquadBase))

	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Println("\nLoading movesets in parallel (with optimizations)...")

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

			var moveWg sync.WaitGroup
			var movesetMutex sync.Mutex

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

			var moveWg sync.WaitGroup
			var movesetMutex sync.Mutex

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

	wg.Wait()

	totalLoadTime := time.Since(totalStartTime)
	fmt.Printf("\nAll movesets loaded successfully! Total loading time: %.2f seconds\n",
		totalLoadTime.Seconds())

	return playerSquad, enemySquad, playerMovesets, enemyMovesets, playerActiveIndex, enemyActiveIndex
}

func SetupMPSquad() ([]*BattlePokemon, []*BattlePokemon, [][]*pokemon.MoveInfo, [][]*pokemon.MoveInfo, int, int) {

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

			time.Sleep(200 * time.Millisecond)
		}

		return nil, err
	}

	playerSquadBase := pokemon.SelectRandSquad()
	enemySquadBase := pokemon.SelectRandSquad()

	playerSelect := 1
	playerActiveIndex := playerSelect

	enemySelect := 1
	enemyActiveIndex := enemySelect

	fmt.Printf("You sent out %s!\n", playerSquadBase[playerSelect].Name)
	fmt.Printf("Enemy sent out %s!\n", enemySquadBase[enemySelect].Name)

	playerSquad := make([]*BattlePokemon, len(playerSquadBase))
	enemySquad := make([]*BattlePokemon, len(enemySquadBase))

	playerMovesets := make([][]*pokemon.MoveInfo, len(playerSquadBase))
	enemyMovesets := make([][]*pokemon.MoveInfo, len(enemySquadBase))

	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Println("\nLoading movesets")

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

			var moveWg sync.WaitGroup
			var movesetMutex sync.Mutex

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

			var moveWg sync.WaitGroup
			var movesetMutex sync.Mutex

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

	wg.Wait()

	return playerSquad, enemySquad, playerMovesets, enemyMovesets, playerActiveIndex, enemyActiveIndex
}
