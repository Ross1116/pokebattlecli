package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
)

func main() {
	start := time.Now()

	playerBattlePokemon, enemyBattlePokemon, playerMoveset, enemyMoveset := battle.SetupGameWithMovesets()
	var selectedMove int

	// Store initial HP values for percentage calculations
	playerMaxHP := playerBattlePokemon.CurrentHP
	enemyMaxHP := enemyBattlePokemon.CurrentHP

	for {
		// Calculate HP percentages
		playerHPPercent := (playerBattlePokemon.CurrentHP / playerMaxHP) * 100
		enemyHPPercent := (enemyBattlePokemon.CurrentHP / enemyMaxHP) * 100

		// Show both Pokémon's HP and stats with percentages
		fmt.Printf("\nYour %s's HP: %.2f/%.2f (%.2f%%)\n",
			playerBattlePokemon.Base.Name, playerBattlePokemon.CurrentHP, playerMaxHP, playerHPPercent)
		fmt.Printf("Enemy %s's HP: %.2f/%.2f (%.2f%%)\n",
			enemyBattlePokemon.Base.Name, enemyBattlePokemon.CurrentHP, enemyMaxHP, enemyHPPercent)

		// Show the moveset and PP for each move
		fmt.Println("\nYour moveset and remaining PP:")
		for i, move := range playerMoveset {
			fmt.Printf("%d. %s (PP: %d)\n", i+1, move.Name, playerBattlePokemon.MovePP[move.Name])
		}

		// User selects a move
		fmt.Print("\nSelect your move (enter a number 1 - 4): ")
		fmt.Scan(&selectedMove)

		// Ensure the move is valid and has remaining PP
		if selectedMove < 1 || selectedMove > len(playerMoveset) {
			fmt.Println("Invalid move selection. Please try again.")
			continue
		}

		// Get the move data for the selected move
		moveData := playerMoveset[selectedMove-1]

		// Ensure the move has PP left
		if playerBattlePokemon.MovePP[moveData.Name] == 0 {
			fmt.Println("This move has no PP left. Please select another move.")
			continue
		}

		// Reduce the PP of the selected move
		if !playerBattlePokemon.UseMove(moveData.Name) {
			fmt.Println("Move failed or has no PP left.")
			continue
		}

		// Display move information
		fmt.Printf("You used: %s, Move accuracy: %d, Move Power: %d\n", moveData.Name, moveData.Accuracy, moveData.Power)

		// Apply damage calculation for the opponent
		damage, percent := battle.DamageCalc(playerBattlePokemon.Base, enemyBattlePokemon.Base, moveData)

		// Apply damage to the opponent
		enemyBattlePokemon.ApplyDamage(damage)

		// Show the result of the attack
		fmt.Printf("You dealt %d damage! (~%.2f%% of %s's HP)\n", damage, percent, enemyBattlePokemon.Base.Name)

		// Check if the opponent has fainted
		if enemyBattlePokemon.Fainted {
			fmt.Printf("\n%s has fainted! You win!\n", enemyBattlePokemon.Base.Name)
			break
		}

		// Opponent's turn to attack (randomly select an opponent move)
		opponentMoveIndex := rand.Intn(len(enemyMoveset))
		opponentMoveData := enemyMoveset[opponentMoveIndex]

		// Make sure the opponent has PP for the move
		if enemyBattlePokemon.MovePP[opponentMoveData.Name] > 0 {
			// Apply opponent's damage to your Pokémon
			enemyBattlePokemon.UseMove(opponentMoveData.Name)
			opponentDamage, opponentPercent := battle.DamageCalc(enemyBattlePokemon.Base, playerBattlePokemon.Base, opponentMoveData)
			playerBattlePokemon.ApplyDamage(opponentDamage)

			// Show the result of the opponent's attack
			fmt.Printf("%s used %s! It dealt %d damage! (~%.2f%% of your Pokémon's HP)\n",
				enemyBattlePokemon.Base.Name, opponentMoveData.Name, opponentDamage, opponentPercent)
		} else {
			fmt.Printf("%s tried to use %s but has no PP left!\n", enemyBattlePokemon.Base.Name, opponentMoveData.Name)
		}

		// Check if your Pokémon has fainted
		if playerBattlePokemon.Fainted {
			fmt.Printf("\nYour %s has fainted! You lose!\n", playerBattlePokemon.Base.Name)
			break
		}

		// Calculate updated HP percentages
		playerHPPercent = (playerBattlePokemon.CurrentHP / playerMaxHP) * 100
		enemyHPPercent = (enemyBattlePokemon.CurrentHP / enemyMaxHP) * 100

		// Display remaining HP of both Pokémon with percentages
		fmt.Printf("\nYour %s's Remaining HP: %.2f/%.2f (%.2f%%)\n",
			playerBattlePokemon.Base.Name, playerBattlePokemon.CurrentHP, playerMaxHP, playerHPPercent)
		fmt.Printf("Enemy %s's Remaining HP: %.2f/%.2f (%.2f%%)\n",
			enemyBattlePokemon.Base.Name, enemyBattlePokemon.CurrentHP, enemyMaxHP, enemyHPPercent)

		// Pause for a moment before the next round
		time.Sleep(1 * time.Second)
	}

	elapsed := time.Since(start)
	fmt.Println("\nExecution Time:", elapsed)
}
