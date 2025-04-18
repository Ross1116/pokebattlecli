package battle

import (
	"fmt"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func DisplayBattleState(playerSquad, enemySquad []*BattlePokemon, playerActiveIndex, enemyActiveIndex int, playerMaxHPs, enemyMaxHPs []float64) {
	fmt.Println("\n=== BATTLEFIELD STATUS ===")
	playerActive := playerSquad[playerActiveIndex]
	enemyActive := enemySquad[enemyActiveIndex]

	playerPercent := (playerActive.CurrentHP / playerMaxHPs[playerActiveIndex]) * 100
	enemyPercent := (enemyActive.CurrentHP / enemyMaxHPs[enemyActiveIndex]) * 100

	fmt.Printf("Your %s - HP: %.1f/%.1f (%.0f%%) - Status: %s\n", playerActive.Base.Name, playerActive.CurrentHP, playerMaxHPs[playerActiveIndex], playerPercent, playerActive.Status)
	fmt.Printf("Enemy %s - HP: %.1f/%.1f (%.0f%%) - Status: %s\n", enemyActive.Base.Name, enemyActive.CurrentHP, enemyMaxHPs[enemyActiveIndex], enemyPercent, enemyActive.Status)
	fmt.Println("==================================")

	fmt.Println("\nYour Team:")
	for i, poke := range playerSquad {
		status := "Ready"
		if poke.Fainted {
			status = "Fainted"
		} else if i == playerActiveIndex {
			status = "Active"
		}
		hpPercent := (poke.CurrentHP / playerMaxHPs[i]) * 100
		fmt.Printf("%d. %s - HP: %.1f/%.1f (%.0f%%) - %s\n", i+1, poke.Base.Name, poke.CurrentHP, playerMaxHPs[i], hpPercent, status)
	}

	fmt.Println("\nOpponent's Team:")
	for i, poke := range enemySquad {
		status := "Ready"
		if poke.Fainted {
			status = "Fainted"
		} else if i == enemyActiveIndex {
			status = "Active"
		}
		hpPercent := (poke.CurrentHP / enemyMaxHPs[i]) * 100
		fmt.Printf("%d. %s - HP: %.1f/%.1f (%.0f%%) - %s\n", i+1, poke.Base.Name, poke.CurrentHP, enemyMaxHPs[i], hpPercent, status)
	}
}

func DisplayMoveOptions(moves []*pokemon.MoveInfo, ppMap map[string]int) {
	fmt.Println("\nMoveset:")
	for i, move := range moves {
		fmt.Printf("%d. %s (PP: %d)\n", i+1, move.Name, ppMap[move.Name])
	}

}

func NextAvailablePokemon(squad []pokemon.Pokemon, currentIndex int) int {
	for i, p := range squad {
		if !p.Fainted && i != currentIndex {
			return i
		}
	}
	return -1
}

func SelectPokemon(squad []pokemon.Pokemon) int {
	for {
		var idx int
		fmt.Print("Choose Pokémon by number: ")
		fmt.Scan(&idx)
		idx--

		if idx < 0 || idx >= len(squad) {
			fmt.Println("Invalid selection.")
			continue
		}

		if squad[idx].Fainted {
			fmt.Println("This Pokémon has fainted.")
			continue
		}

		return idx
	}
}
