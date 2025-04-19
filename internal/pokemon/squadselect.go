package pokemon

import (
	"fmt"
	"math/rand"
	"sync"
)

func SelectRandSquad() []*Pokemon {
	var squad []*Pokemon
	var wg sync.WaitGroup
	var mu sync.Mutex
	uniquePokemon := make(map[int]struct{})

	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			var poke *Pokemon
			var randNum int
			for {
				randNum = rand.Intn(386) + 1
				if _, exists := uniquePokemon[randNum]; !exists {
					break
				}
			}

			fmt.Printf("Fetching Pokémon #%d...\n", randNum)
			poke, err := FetchPokemonData(randNum)
			if err != nil {
				mu.Lock()
				fmt.Printf("Error fetching Pokémon #%d: %v\n", randNum, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			uniquePokemon[randNum] = struct{}{}
			fmt.Printf("Completed fetching Pokémon #%d: %s\n", randNum, poke.Name)
			mu.Unlock()

			mu.Lock()
			squad = append(squad, poke)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	fmt.Println("")
	return squad
}
