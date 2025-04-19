package battle

import (
	"fmt"
	"math/rand/v2"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func ResolveTurn(player *BattlePokemon, enemy *BattlePokemon, playerMove *pokemon.MoveInfo, enemyMove *pokemon.MoveInfo) (*BattlePokemon, *BattlePokemon, *pokemon.MoveInfo, *pokemon.MoveInfo) {
	if isSwitching(playerMove) {
		return player, enemy, nil, nil
	} else if isSwitching(enemyMove) {
		return enemy, player, nil, nil
	}

	playerPriority := getMovePriority(playerMove)
	enemyPriority := getMovePriority(enemyMove)

	if playerPriority == enemyPriority {
		playerSpeed := stats.StatCalc(stats.GetStat(player.Base, "speed"))
		enemySpeed := stats.StatCalc(stats.GetStat(enemy.Base, "speed"))

		if playerSpeed == enemySpeed {
			if rand.Float64() < 0.5 {
				return player, enemy, playerMove, enemyMove
			} else {
				return enemy, player, enemyMove, playerMove
			}
		} else if playerSpeed > enemySpeed {
			return player, enemy, playerMove, enemyMove
		} else {
			return enemy, player, enemyMove, playerMove
		}
	}

	if playerPriority > enemyPriority {
		return player, enemy, playerMove, enemyMove
	} else {
		return enemy, player, enemyMove, playerMove
	}
}

func isSwitching(move *pokemon.MoveInfo) bool {
	return move != nil && move.Name == "Switch"
}

func getMovePriority(move *pokemon.MoveInfo) int {
	if move != nil {
		return move.Priority
	}
	return 0
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
