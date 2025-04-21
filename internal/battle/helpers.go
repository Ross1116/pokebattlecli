package battle

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func EnemyAttack(attacker, defender *BattlePokemon, moveSet []*pokemon.MoveInfo) []string {
	events := []string{}
	if attacker == nil || defender == nil || len(moveSet) == 0 || attacker.Fainted {
		return events
	}

	opponentMoveIndex := rand.IntN(len(moveSet))
	opponentMoveData := moveSet[opponentMoveIndex]
	if opponentMoveData == nil {
		events = append(events, fmt.Sprintf("%s has an invalid move!", attacker.Base.Name))
		return events
	}

	if attacker.UseMove(opponentMoveData.Name) {
		dmg, percent, calcEvents := DamageCalc(attacker, defender, opponentMoveData)
		events = append(events, calcEvents...)

		if dmg > 0 {
			defender.ApplyDamage(float64(dmg))
			events = append(events, fmt.Sprintf("%s used %s!", attacker.Base.Name, opponentMoveData.Name))
			events = append(events, fmt.Sprintf("%s took %d damage! (~%.1f%%)", defender.Base.Name, dmg, percent))
			if defender.Fainted {
				events = append(events, fmt.Sprintf("%s fainted!", defender.Base.Name))
			}
		} else if !containsMiss(calcEvents) && !containsImmune(calcEvents) {
			events = append(events, fmt.Sprintf("%s used %s!", attacker.Base.Name, opponentMoveData.Name))
			if effectivenessCheck(opponentMoveData, defender) > 0 {
				events = append(events, fmt.Sprintf("But it had no effect on %s!", defender.Base.Name))
			}
		} else {
			events = append(events, fmt.Sprintf("%s used %s!", attacker.Base.Name, opponentMoveData.Name))
		}
	} else {
		events = append(events, fmt.Sprintf("%s tried to use %s but has no PP left!", attacker.Base.Name, opponentMoveData.Name))
	}
	return events
}

func containsMiss(events []string) bool {
	for _, e := range events {
		if strings.Contains(e, "missed!") {
			return true
		}
	}
	return false
}
func containsImmune(events []string) bool {
	for _, e := range events {
		if strings.Contains(e, "doesn't affect") {
			return true
		}
	}
	return false
}

func DisplayBattleStatus(player, enemy *BattlePokemon) {
	if player == nil || enemy == nil || player.Base == nil || enemy.Base == nil {
		return
	}

	playerMaxHP := 0.0
	if player.Base.Stats != nil {
		for _, s := range player.Base.Stats {
			if s.Stat.Name == "hp" {
				playerMaxHP = stats.HpCalc(s.BaseStat)
				break
			}
		}
	}
	enemyMaxHP := 0.0
	if enemy.Base.Stats != nil {
		for _, s := range enemy.Base.Stats {
			if s.Stat.Name == "hp" {
				enemyMaxHP = stats.HpCalc(s.BaseStat)
				break
			}
		}
	}

	playerHPPercent := 0.0
	if playerMaxHP > 0 {
		playerHPPercent = (player.CurrentHP / playerMaxHP) * 100
	}
	enemyHPPercent := 0.0
	if enemyMaxHP > 0 {
		enemyHPPercent = (enemy.CurrentHP / enemyMaxHP) * 100
	}

	fmt.Printf("\nYour %s's HP: %.1f/%.1f (%.0f%%)", player.Base.Name, player.CurrentHP, playerMaxHP, playerHPPercent)
	if player.Status != "" {
		fmt.Printf(" [%s]", player.Status)
	}
	fmt.Println()
	fmt.Printf("Enemy %s's HP: %.1f/%.1f (%.0f%%)", enemy.Base.Name, enemy.CurrentHP, enemyMaxHP, enemyHPPercent)
	if enemy.Status != "" {
		fmt.Printf(" [%s]", enemy.Status)
	}
	fmt.Println()
}

func effectivenessCheck(move *pokemon.MoveInfo, defender *BattlePokemon) float64 {
	effectiveness := 1.0
	if defender.Base != nil && defender.Base.Types != nil && move != nil && pokemon.TypeEffectiveness != nil {
		for _, t := range defender.Base.Types {
			if attackerEffectiveness, ok := pokemon.TypeEffectiveness[move.Type.Name]; ok {
				if mult, ok2 := attackerEffectiveness[t.Type.Name]; ok2 {
					effectiveness *= mult
				}
			}
		}
	}
	return effectiveness
}

