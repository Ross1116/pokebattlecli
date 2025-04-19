package battle

import (
	"fmt"
	"math/rand/v2"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func ResolveTurn(player *BattlePokemon, enemy *BattlePokemon, playerMoves []*pokemon.MoveInfo, enemyMoves []*pokemon.MoveInfo) (*BattlePokemon, *BattlePokemon, *pokemon.MoveInfo, *pokemon.MoveInfo) {
	if playerSwitching := isSwitching(playerMoves); playerSwitching {
		return player, enemy, nil, nil
	} else if enemySwitching := isSwitching(enemyMoves); enemySwitching {
		return enemy, player, nil, nil
	}

	playerPriority := getMovePriority(player, playerMoves)
	enemyPriority := getMovePriority(enemy, enemyMoves)

	if playerPriority == enemyPriority {
		playerSpeed := stats.StatCalc(stats.GetStat(player.Base, "speed"))
		enemySpeed := stats.StatCalc(stats.GetStat(enemy.Base, "speed"))

		if playerSpeed == enemySpeed {
			if rand.Float64() < 0.5 {
				return player, enemy, playerMoves[0], enemyMoves[0]
			} else {
				return enemy, player, enemyMoves[0], playerMoves[0]
			}
		} else if playerSpeed > enemySpeed {
			return player, enemy, playerMoves[0], enemyMoves[0]
		} else {
			return enemy, player, enemyMoves[0], playerMoves[0]
		}
	}

	if playerPriority > enemyPriority {
		return player, enemy, playerMoves[0], enemyMoves[0]
	} else {
		return enemy, player, enemyMoves[0], playerMoves[0]
	}
}

func isSwitching(moves []*pokemon.MoveInfo) bool {
	for _, move := range moves {
		if move.Name == "Switch" {
			return true
		}
	}
	return false
}

func getMovePriority(pokemon *BattlePokemon, moves []*pokemon.MoveInfo) int {
	highestPriority := 0
	for _, move := range moves {
		if move.Priority > highestPriority {
			highestPriority = move.Priority
		}
	}
	return highestPriority
}

func (bp *BattlePokemon) HandleTurnEffects() {
	switch bp.Status {
	case "burn":
		dmg := stats.HpCalc(stats.GetStat(bp.Base, "hp")) / 16
		bp.ApplyDamage(dmg)
	case "poison":
		dmg := stats.HpCalc(stats.GetStat(bp.Base, "hp")) / 8
		bp.ApplyDamage(dmg)
	case "toxic":
		stacks := 1 + (5 - bp.StatusTurns)
		dmg := float64(stats.HpCalc(stats.GetStat(bp.Base, "hp"))) * float64(stacks) / 16
		bp.ApplyDamage(dmg)
	case "sleep", "freeze":
		bp.StatusTurns--
		if bp.StatusTurns <= 0 {
			bp.Status = ""
		}
	}

	if bp.Volatile["confusion"] {
		bp.StatusTurns--
		if bp.StatusTurns <= 0 {
			bp.RemoveVolatileEffect("confusion")
		}
	}
}

func (bp *BattlePokemon) CanAct() bool {
	if bp.Status == "sleep" || bp.Status == "freeze" {
		return false
	}
	if bp.Volatile["flinch"] {
		bp.RemoveVolatileEffect("flinch")
		return false
	}
	if bp.Status == "paralysis" {
		if rand.Float64() < 0.25 {
			return false
		}
	}
	if bp.Volatile["confusion"] {
		if rand.Float64() < 0.33 {
			dmg := stats.HpCalc(stats.GetStat(bp.Base, "hp")) / 8
			bp.ApplyDamage(dmg)
			return false
		}
	}
	return true
}

func ProcessPlayerTurn(player *BattlePokemon, enemy *BattlePokemon, move *pokemon.MoveInfo) {
	if !player.CanAct() {
		fmt.Println("Player cannot act this turn!")
		return
	}

	if player.UseMove(move.Name) {
		dmg, percent := DamageCalc(player.Base, enemy.Base, move)
		enemy.ApplyDamage(float64(dmg))
		fmt.Printf("\n\033[1mYou dealt %d damage (~%.2f%% of enemy HP) with %s!\033[0m\n", dmg, percent, move.Name)
	}
}

func ProcessEnemyTurn(player *BattlePokemon, enemy *BattlePokemon, move *pokemon.MoveInfo) {
	if !enemy.CanAct() {
		fmt.Println("Enemy cannot act this turn!")
		return
	}

	if enemy.UseMove(move.Name) {
		dmg, percent := DamageCalc(enemy.Base, player.Base, move)
		player.ApplyDamage(float64(dmg))
		fmt.Printf("\n\033[1mEnemy dealt %d damage (~%.2f%% of your HP) with %s!\033[0m\n", dmg, percent, move.Name)
	}
}
