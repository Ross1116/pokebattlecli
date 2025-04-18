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

	playerSquad, enemySquad, playerMovesets, enemyMovesets := battle.SetupFullSquads()

	playerActiveIndex := 0
	enemyActiveIndex := 0

	playerMaxHPs := make([]float64, len(playerSquad))
	enemyMaxHPs := make([]float64, len(enemySquad))
	for i, p := range playerSquad {
		playerMaxHPs[i] = p.CurrentHP
	}
	for i, p := range enemySquad {
		enemyMaxHPs[i] = p.CurrentHP
	}

	var playerPokemonSquad []pokemon.Pokemon
	var enemyPokemonSquad []pokemon.Pokemon

	for _, bp := range playerSquad {
		playerPokemonSquad = append(playerPokemonSquad, *bp.Base)
	}
	for _, bp := range enemySquad {
		enemyPokemonSquad = append(enemyPokemonSquad, *bp.Base)
	}

	for {
		battle.DisplayBattleState(playerSquad, enemySquad, playerActiveIndex, enemyActiveIndex, playerMaxHPs, enemyMaxHPs)

		battle.DisplayMoveOptions(playerMovesets[playerActiveIndex], playerSquad[playerActiveIndex].MovePP)
		fmt.Println("0. Switch Pokémon")
		fmt.Print("Select your action (0-4): ")

		var choice int
		fmt.Scan(&choice)

		if choice == 0 {
			fmt.Println("Select Pokémon to switch to:")
			for i, p := range playerSquad {
				fmt.Printf("%d. %s (HP: %d%%)\n", i+1, p.Base.Name, int((p.CurrentHP/playerMaxHPs[i])*100))
			}
			fmt.Println("0. Go back to move selection (Cancel switch)")
			fmt.Print("Enter your choice (0-6): ")

			var switchChoice int
			fmt.Scan(&switchChoice)

			if switchChoice == 0 {
				continue
			}

			if switchChoice >= 1 && switchChoice <= len(playerSquad) {
				if switchChoice-1 == playerActiveIndex {
					fmt.Println("This Pokémon is already active.")
					continue
				}

				playerActiveIndex = switchChoice - 1
				fmt.Printf("You switched to %s!\n", playerSquad[playerActiveIndex].Base.Name)
				continue
			} else {
				fmt.Println("Invalid choice. Please select a valid Pokémon.")
				continue
			}
		}

		if choice < 1 || choice > 4 {
			fmt.Println("Invalid choice. Please select a valid action.")
			continue
		}

		moveData := playerMovesets[playerActiveIndex][choice-1]

		if playerSquad[playerActiveIndex].MovePP[moveData.Name] == 0 {
			fmt.Println("This move has no PP left.")
			continue
		}

		if !playerSquad[playerActiveIndex].UseMove(moveData.Name) {
			fmt.Println("Move failed.")
			continue
		}

		fmt.Printf("You used %s!\n", moveData.Name)
		damage, percent := battle.DamageCalc(playerSquad[playerActiveIndex].Base, enemySquad[enemyActiveIndex].Base, moveData)
		enemySquad[enemyActiveIndex].ApplyDamage(damage)
		fmt.Printf("Dealt %d damage (~%.2f%% of HP).\n", damage, percent)

		if enemySquad[enemyActiveIndex].Fainted {
			newEnemyIndex := battle.NextAvailablePokemon(enemyPokemonSquad, enemyActiveIndex)
			if newEnemyIndex == -1 {
				fmt.Println("\nAll enemy Pokémon have fainted! You win!")
				break
			}
			enemyActiveIndex = newEnemyIndex
			fmt.Printf("Enemy sent out %s!\n", enemySquad[enemyActiveIndex].Base.Name)
			continue
		}

		if rand.Float64() < 0.2 {
			newEnemyIndex := battle.NextAvailablePokemon(enemyPokemonSquad, enemyActiveIndex)
			if newEnemyIndex != -1 {
				enemyActiveIndex = newEnemyIndex
				enemyBattlePokemon := enemySquad[enemyActiveIndex]
				fmt.Printf("Enemy switched to %s!\n", enemyBattlePokemon.Base.Name)
			}
		} else {
			battle.EnemyAttack(enemySquad[enemyActiveIndex], playerSquad[playerActiveIndex], enemyMovesets[enemyActiveIndex])
		}

		if playerSquad[playerActiveIndex].Fainted {
			fmt.Printf("\nYour %s has fainted. Choose a replacement.\n", playerSquad[playerActiveIndex].Base.Name)
			newIndex := battle.SelectPokemon(playerPokemonSquad)
			playerActiveIndex = newIndex
			fmt.Printf("You sent out %s!\n", playerSquad[playerActiveIndex].Base.Name)
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nExecution Time:", time.Since(start))
}

