package battle

import (
	"fmt"
	"math/rand/v2"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func ResolveTurn(player *BattlePokemon, enemy *BattlePokemon, playerMove *pokemon.MoveInfo, enemyMove *pokemon.MoveInfo) (*BattlePokemon, *BattlePokemon, *pokemon.MoveInfo, *pokemon.MoveInfo) {
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

func getMovePriority(move *pokemon.MoveInfo) int {
	if move != nil {
		return move.Priority
	}
	return 0
}

func (bp *BattlePokemon) HandleTurnEffects() []string {
	events := []string{}
	if bp.Fainted {
		return events
	}
	maxHP := 0.0
	if bp.Base != nil && bp.Base.Stats != nil {
		for _, stat := range bp.Base.Stats {
			if stat.Stat.Name == "hp" {
				maxHP = stats.HpCalc(stat.BaseStat)
				break
			}
		}
	}

	statusDamage := 0.0
	statusMsg := ""
	switch bp.Status {
	case "brn":
		if maxHP > 0 {
			statusDamage = maxHP / 16.0
		}
		statusMsg = fmt.Sprintf("%s took damage from its burn!", bp.Base.Name)
	case "psn":
		if maxHP > 0 {
			statusDamage = maxHP / 8.0
		}
		statusMsg = fmt.Sprintf("%s took damage from poison!", bp.Base.Name)
	case "tox":
		stacks := 1
		if bp.StatusTurns > 0 {
			stacks = bp.StatusTurns
		}
		if maxHP > 0 {
			statusDamage = float64(stacks) * maxHP / 16.0
		}
		statusMsg = fmt.Sprintf("%s took heavy damage from poison!", bp.Base.Name)
		bp.StatusTurns++
	}
	if statusDamage > 0 {
		bp.ApplyDamage(statusDamage)
		events = append(events, statusMsg)
		if bp.Fainted {
			events = append(events, fmt.Sprintf("%s fainted!", bp.Base.Name))
			return events
		}
	}

	switch bp.Status {
	case "slp":
		if bp.StatusTurns > 0 {
			bp.StatusTurns--
			if bp.StatusTurns == 0 {
				bp.Status = ""
				events = append(events, fmt.Sprintf("%s woke up!", bp.Base.Name))
			}
		} else {
			bp.Status = ""
			events = append(events, fmt.Sprintf("%s woke up! (Forced)", bp.Base.Name))
		}
	case "frz":
		break
	}
	return events
}

func (bp *BattlePokemon) CanAct() (bool, []string) {
	events := []string{}
	if bp.Fainted {
		return false, events
	}
	if bp.Volatile["flinch"] {
		events = append(events, fmt.Sprintf("%s flinched and couldn't move!", bp.Base.Name))
		bp.RemoveVolatileEffect("flinch")
		return false, events
	}
	switch bp.Status {
	case "slp":
		if bp.StatusTurns > 0 {
			events = append(events, fmt.Sprintf("%s is fast asleep.", bp.Base.Name))
			return false, events
		} else {
			bp.Status = ""
			events = append(events, fmt.Sprintf("%s woke up!", bp.Base.Name))
		}
	case "frz":
		if rand.Float64() < 0.2 {
			bp.Status = ""
			events = append(events, fmt.Sprintf("%s thawed out!", bp.Base.Name))
		} else {
			events = append(events, fmt.Sprintf("%s is frozen solid!", bp.Base.Name))
			return false, events
		}
	case "par":
		if rand.Float64() < 0.25 {
			events = append(events, fmt.Sprintf("%s is paralyzed! It can't move!", bp.Base.Name))
			return false, events
		}
	}
	if bp.Volatile["confusion"] {
		events = append(events, fmt.Sprintf("%s is confused!", bp.Base.Name))
		if rand.Float64() < 0.33 {
			events = append(events, "It hurt itself in its confusion!")
			maxHP := 0.0
			hpStatFound := false
			if bp.Base != nil && bp.Base.Stats != nil {
				for _, stat := range bp.Base.Stats {
					if stat.Stat.Name == "hp" {
						maxHP = stats.HpCalc(stat.BaseStat)
						hpStatFound = true
						break
					}
				}
			}
			if hpStatFound && maxHP > 0 {
				dmg := maxHP / 16.0
				bp.ApplyDamage(dmg)
				if bp.Fainted {
					events = append(events, fmt.Sprintf("%s fainted!", bp.Base.Name))
				}
			} else {
				bp.ApplyDamage(10)
				events = append(events, "(Took confusion damage)")
			}
			return false, events
		}
	}
	return true, events
}

func ProcessPlayerTurn(player *BattlePokemon, enemy *BattlePokemon, move *pokemon.MoveInfo) []string {
	return processAction(player, enemy, move)
}

func ProcessEnemyTurn(player *BattlePokemon, enemy *BattlePokemon, move *pokemon.MoveInfo) []string {
	return processAction(enemy, player, move)
}

func processAction(attacker *BattlePokemon, defender *BattlePokemon, move *pokemon.MoveInfo) []string {
	events := []string{}
	if attacker == nil || defender == nil || move == nil || attacker.Fainted {
		return events
	}
	canAct, preEvents := attacker.CanAct()
	events = append(events, preEvents...)
	if !canAct {
		return events
	}
	if !attacker.UseMove(move.Name) {
		events = append(events, fmt.Sprintf("%s has no PP left for %s!", attacker.Base.Name, move.Name))
		return events
	}
	events = append(events, fmt.Sprintf("%s used %s!", attacker.Base.Name, move.Name))

	if move.Power > 0 {
		dmg, percent, calcEvents := DamageCalc(attacker, defender, move)
		events = append(events, calcEvents...)
		if dmg > 0 {
			defender.ApplyDamage(float64(dmg))
			events = append(events, fmt.Sprintf("%s took %d damage! (%.1f%%)", defender.Base.Name, dmg, percent))
			if defender.Fainted {
				events = append(events, fmt.Sprintf("%s fainted!", defender.Base.Name))
			}
		} else if len(calcEvents) == 0 && effectivenessCheck(move, defender) > 0 {
			events = append(events, fmt.Sprintf("It had no effect on %s!", defender.Base.Name))
		}
	} else {
		events = append(events, fmt.Sprintf("...%s used a 0 damage move %s with effects %s...", attacker.Base.Name, move.Name, move.EffectEntries))
	}
	return events
}

// func effectivenessCheck(move *pokemon.MoveInfo, defender *BattlePokemon) float64 {
// 	effectiveness := 1.0
// 	if defender.Base != nil && defender.Base.Types != nil && move != nil && pokemon.TypeEffectiveness != nil {
// 		for _, t := range defender.Base.Types {
// 			if attackerEffectiveness, ok := pokemon.TypeEffectiveness[move.Type.Name]; ok {
// 				if mult, ok2 := attackerEffectiveness[t.Type.Name]; ok2 {
// 					effectiveness *= mult
// 				}
// 			}
// 		}
// 	}
// 	return effectiveness
// }
