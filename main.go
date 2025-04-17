package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ross1116/pokebattlecli/internal/battle"
)

func main() {
	start := time.Now()

	playerSquad, enemySquad, playerMovesets, enemyMovesets := battle.SetupFullSquads()

	playerActiveIndex := 0
	enemyActiveIndex := 0

	playerBattlePokemon := playerSquad[playerActiveIndex]
	enemyBattlePokemon := enemySquad[enemyActiveIndex]

	playerMaxHPs := make([]float64, len(playerSquad))
	enemyMaxHPs := make([]float64, len(enemySquad))

	for i, p := range playerSquad {
		playerMaxHPs[i] = p.CurrentHP
	}

	for i, p := range enemySquad {
		enemyMaxHPs[i] = p.CurrentHP
	}

	for {
		battle.DisplayBattleStatus(playerBattlePokemon, enemyBattlePokemon, playerMaxHPs[playerActiveIndex], enemyMaxHPs[enemyActiveIndex])

		fmt.Println("\n=== OPPONENT'S TEAM ===")
		for i, poke := range enemySquad {
			status := ""
			if poke.Fainted {
				status = "Fainted"
			} else if i == enemyActiveIndex {
				status = "Active"
			} else {
				status = "Ready"
			}

			if poke.Status != "" {
				status += " (" + poke.Status + ")"
			}

			hpInfo := ""
			if !poke.Fainted {
				hpPercent := (poke.CurrentHP / enemyMaxHPs[i]) * 100
				hpInfo = fmt.Sprintf("HP: %.2f/%.2f (%.2f%%)", poke.CurrentHP, enemyMaxHPs[i], hpPercent)
			}

			fmt.Printf("%d. %s - %s - Status: %s\n", i+1, poke.Base.Name, hpInfo, status)
		}
		fmt.Println("====================")

		fmt.Println("\nWhat would you like to do?")
		fmt.Println("1. Fight")
		fmt.Println("2. Switch Pokémon")

		var choice int
		fmt.Print("Enter your choice (1-2): ")
		fmt.Scan(&choice)

		if choice == 1 {
			fmt.Println("\nYour moveset and remaining PP:")
			for i, move := range playerMovesets[playerActiveIndex] {
				fmt.Printf("%d. %s (PP: %d)\n", i+1, move.Name, playerBattlePokemon.MovePP[move.Name])
			}

			var selectedMove int
			fmt.Print("\nSelect your move (enter a number 1 - 4): ")
			fmt.Scan(&selectedMove)

			if selectedMove < 1 || selectedMove > len(playerMovesets[playerActiveIndex]) {
				fmt.Println("Invalid move selection. Please try again.")
				continue
			}

			moveData := playerMovesets[playerActiveIndex][selectedMove-1]

			if playerBattlePokemon.MovePP[moveData.Name] == 0 {
				fmt.Println("This move has no PP left. Please select another move.")
				continue
			}

			if !playerBattlePokemon.UseMove(moveData.Name) {
				fmt.Println("Move failed or has no PP left.")
				continue
			}

			fmt.Printf("You used: %s, Move accuracy: %d, Move Power: %d\n",
				moveData.Name, moveData.Accuracy, moveData.Power)

			damage, percent := battle.DamageCalc(playerBattlePokemon.Base, enemyBattlePokemon.Base, moveData)
			enemyBattlePokemon.ApplyDamage(damage)

			fmt.Printf("You dealt %d damage! (~%.2f%% of %s's HP)\n",
				damage, percent, enemyBattlePokemon.Base.Name)

		} else if choice == 2 {
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
				newIndex--

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

			playerActiveIndex = newIndex
			playerBattlePokemon = playerSquad[playerActiveIndex]
			fmt.Printf("You switched to %s!\n", playerBattlePokemon.Base.Name)
		} else {
			fmt.Println("Invalid choice. Please try again.")
			continue
		}

		if enemyBattlePokemon.Fainted {
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
			continue
		}

		if rand.Float64() < 0.2 {
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
				battle.EnemyAttack(enemyBattlePokemon, playerBattlePokemon, enemyMovesets[enemyActiveIndex])
			}
		} else {
			battle.EnemyAttack(enemyBattlePokemon, playerBattlePokemon, enemyMovesets[enemyActiveIndex])
		}

		if playerBattlePokemon.Fainted {
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
				newIndex--

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
