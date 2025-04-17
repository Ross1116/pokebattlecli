package battle

import (
	"fmt"
	"math/rand"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func SetupGameWithMovesets() (*BattlePokemon, *BattlePokemon, []*pokemon.MoveInfo, []*pokemon.MoveInfo) {
	playerSquad := pokemon.SelectRandSquad()
	enemySquad := pokemon.SelectRandSquad()

	fmt.Println("Your randomly selected pokemon squad is:")
	for i := range playerSquad {
		fmt.Println(i+1, playerSquad[i].Name)
	}

	fmt.Println("\nEnemy randomly selected pokemon squad is:")
	for i := range enemySquad {
		fmt.Println(i+1, enemySquad[i].Name)
	}

	var playerSelect int
	var enemySelect int

	fmt.Print("Pick your pokemon to fight (enter a number 1 - 6): ")
	fmt.Scan(&playerSelect)

	enemySelect = rand.Intn(6) + 1

	playerPokemon := playerSquad[playerSelect-1]
	enemyPokemon := enemySquad[enemySelect-1]

	fmt.Println("You have selected", playerPokemon.Name)
	fmt.Println("Enemy has selected", enemyPokemon.Name)

	playerApiMoves := pokemon.PickRandMoves(playerPokemon)
	playerMoveset := []*pokemon.MoveInfo{} // This will hold []*pokemon.MoveInfo

	fmt.Println("Fetching your moveset...")
	for _, moveAPI := range playerApiMoves {
		moveData, err := pokemon.FetchMoveData(moveAPI.URL)
		if err != nil {
			fmt.Println("Error fetching move data:", err)
			continue
		}
		playerMoveset = append(playerMoveset, moveData)
	}

	enemyApiMoves := pokemon.PickRandMoves(enemyPokemon)
	enemyMoveset := []*pokemon.MoveInfo{} // This will hold []*pokemon.MoveInfo

	fmt.Println("Fetching enemy moveset...")
	for _, moveAPI := range enemyApiMoves {
		moveData, err := pokemon.FetchMoveData(moveAPI.URL)
		if err != nil {
			fmt.Println("Error fetching move data:", err)
			continue
		}
		enemyMoveset = append(enemyMoveset, moveData)
	}

	playerBattlePokemon := NewBattlePokemon(playerPokemon, playerMoveset)
	enemyBattlePokemon := NewBattlePokemon(enemyPokemon, enemyMoveset)

	return playerBattlePokemon, enemyBattlePokemon, playerMoveset, enemyMoveset
}
