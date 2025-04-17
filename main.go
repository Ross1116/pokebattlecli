package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func main() {
	start := time.Now()

	// Select a random squad of 6 Pokémon
	squad := pokemon.SelectRandSquad()
	var selectedPokemon int
	var selectedMove int

	fmt.Println("Your randomly selected pokemon squad is:")
	for i := range squad {
		fmt.Println(i+1, squad[i].Name)
	}

	// User selects a Pokémon
	fmt.Print("Pick your pokemon to fight (enter a number 1 - 6): ")
	fmt.Scan(&selectedPokemon)

	selectedPoke := squad[selectedPokemon-1]
	fmt.Println("Your selected pokemon is", selectedPoke.Name)

	// Select random moves for the selected Pokémon
	apiMoves := pokemon.PickRandMoves(selectedPoke) // This returns []pokemon.ApiResource
	moveset := []*pokemon.MoveInfo{}                // This will hold []*pokemon.MoveInfo

	// Convert ApiResource to MoveInfo by fetching data from URLs
	for _, moveAPI := range apiMoves {
		moveData, err := pokemon.FetchMoveData(moveAPI.URL)
		if err != nil {
			fmt.Println("Error fetching move data:", err)
			continue
		}
		moveset = append(moveset, moveData)
	}

	// Initialize the battle state for the selected Pokémon
	battlePokemon := battle.NewBattlePokemon(selectedPoke, moveset)

	// Initialize the opposing Pokémon
	opponent := squad[0] // or any other Pokémon from the squad

	// Initialize the battle state for the opponent
	opponentBattle := battle.NewBattlePokemon(opponent, moveset)

	// Infinite loop until one of the Pokémon faints
	for {
		// Show both Pokémon's HP and stats
		fmt.Printf("\nYour Pokémon's HP: %.2f\n", battlePokemon.CurrentHP)
		fmt.Printf("%s's HP: %.2f\n", opponent.Name, opponentBattle.CurrentHP)

		// Show the moveset and PP for each move
		fmt.Println("\nYour moveset and remaining PP:")
		for i, move := range moveset {
			fmt.Printf("%d. %s (PP: %d)\n", i+1, move.Name, battlePokemon.MovePP[move.Name])
		}

		// User selects a move
		fmt.Print("\nSelect your move (enter a number 1 - 4): ")
		fmt.Scan(&selectedMove)

		// Ensure the move is valid and has remaining PP
		if battlePokemon.MovePP[moveset[selectedMove-1].Name] == 0 {
			fmt.Println("This move has no PP left. Please select another move.")
			continue
		}

		// Get the move data for the selected move
		moveData := moveset[selectedMove-1]

		// Reduce the PP of the selected move
		if !battlePokemon.UseMove(moveData.Name) {
			fmt.Println("Move failed or has no PP left.")
			continue
		}

		// Display move information
		fmt.Printf("You used: %s, Move accuracy: %d, Move Power: %d\n", moveData.Name, moveData.Accuracy, moveData.Power)

		// Apply damage calculation for the opponent
		damage, percent := battle.DamageCalc(battlePokemon.Base, opponentBattle.Base, moveData)

		// Apply damage to the opponent
		opponentBattle.ApplyDamage(damage)

		// Show the result of the attack
		fmt.Printf("You dealt %d damage! (~%.2f%% of %s's HP)\n", damage, percent, opponent.Name)

		// Check if the opponent has fainted
		if opponentBattle.Fainted {
			fmt.Printf("\n%s has fainted! You win!\n", opponent.Name)
			break
		}

		// Opponent's turn to attack (randomly select an opponent move for now)
		opponentMoveIndex := rand.Intn(len(moveset))
		opponentMoveData := moveset[opponentMoveIndex]

		// Apply opponent's damage to your Pokémon
		opponentDamage, opponentPercent := battle.DamageCalc(opponentBattle.Base, battlePokemon.Base, opponentMoveData)
		battlePokemon.ApplyDamage(opponentDamage)

		// Show the result of the opponent's attack
		fmt.Printf("%s used %s! It dealt %d damage! (~%.2f%% of your Pokémon's HP)\n", opponent.Name, opponentMoveData.Name, opponentDamage, opponentPercent)

		// Check if your Pokémon has fainted
		if battlePokemon.Fainted {
			fmt.Printf("\nYour Pokémon has fainted! You lose!\n")
			break
		}

		// Display remaining HP of both Pokémon
		fmt.Printf("\nYour Pokémon's Remaining HP: %.2f\n", battlePokemon.CurrentHP)
		fmt.Printf("%s's Remaining HP: %.2f\n", opponent.Name, opponentBattle.CurrentHP)

		// Pause for a moment before the next round (optional)
		time.Sleep(1 * time.Second)
	}

	elapsed := time.Since(start)
	fmt.Println("\nExecution Time:", elapsed)
}
