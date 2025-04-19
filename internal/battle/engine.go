package battle

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func DamageCalc(attacker *pokemon.Pokemon, defender *pokemon.Pokemon, move *pokemon.MoveInfo) (int, float64) {
	level := 100
	var atkStat, defStat int

	switch move.DamageClass.Name {
	case "physical":
		atkStat = int(stats.StatCalc(stats.GetStat(attacker, "attack")))
		defStat = int(stats.StatCalc(stats.GetStat(defender, "defense")))

	case "special":
		atkStat = int(stats.StatCalc(stats.GetStat(attacker, "special-attack")))
		defStat = int(stats.StatCalc(stats.GetStat(defender, "special-defense")))
	default:
		log.Printf("Unsupported move class: %s", move.DamageClass.Name)
		return 0, 0
	}

	if atkStat == 0 || defStat == 0 {
		log.Printf("Stat retrieval failed. atk: %d, def: %d", atkStat, defStat)
		return 0, 0
	}

	if move.Power == 0 {
		log.Printf("Move has zero power: %s", move.Name)
		return 0, 0
	}

	stab := 1.0
	for _, t := range attacker.Types {
		if t.Type.Name == move.Type.Name {
			stab = 1.5
			fmt.Printf("\nSTAB Bonus %.2f\n", stab)
			break
		}
	}

	effectiveness := 1.0
	for _, t := range defender.Types {
		if mult, ok := pokemon.TypeEffectiveness[move.Type.Name][t.Type.Name]; ok {
			effectiveness *= mult
		}
	}
	fmt.Printf("\nType Effectiveness %.2f\n", effectiveness)

	acc := float64(move.Accuracy)
	if acc == 0 {
		acc = 100
	}

	hitRoll := rand.Float64() * 100
	if hitRoll > acc {
		log.Printf("The move %s missed! Accuracy roll: %.2f > %.2f\n", move.Name, hitRoll, acc)
		return 0, 0
	}

	baseDmg := (((2*float64(level)/5 + 2) * float64(move.Power) * float64(atkStat) / float64(defStat)) / 50) + 2
	randomFactor := 0.85 + (rand.Float64() * 0.15)
	finalDmg := baseDmg * stab * effectiveness * randomFactor

	critChance := rand.Float64() * 100
	wasCrit := critChance < 6.25
	if critChance < 6.25 {
		fmt.Println("Critical hit!")
		finalDmg *= 1.5
	}

	fmt.Printf("\nDEBUG: Atk:%d | Def: %d | Power: %d | Speed: %d | Priority: %d | BaseDmg: %.2f | STAB: %.2f | TypeEff: %.2f | Random: %.2f | Crit: %v | FinalDmg: %.2f\n",
		atkStat, defStat, move.Power, int(stats.StatCalc(stats.GetStat(attacker, "speed"))), move.Priority, baseDmg, stab, effectiveness, randomFactor, wasCrit, finalDmg)

	rounded := int(math.Floor(finalDmg))
	baseHp := stats.GetStat(defender, "hp")
	totalHp := stats.HpCalc(baseHp)
	percent := (finalDmg / totalHp) * 100

	return rounded, percent
}

func ExecuteBattleTurn(player *BattlePokemon, enemy *BattlePokemon, playerMove *pokemon.MoveInfo, enemyMove *pokemon.MoveInfo) {
	first, second, firstMove, secondMove := ResolveTurn(player, enemy, playerMove, enemyMove)

	first.HandleTurnEffects()
	second.HandleTurnEffects()

	if first == player {
		ProcessPlayerTurn(player, enemy, firstMove)
		if enemy.Fainted {
			return
		}
		ProcessEnemyTurn(player, enemy, secondMove)
	} else {
		ProcessEnemyTurn(player, enemy, firstMove)
		if player.Fainted {
			return
		}
		ProcessPlayerTurn(player, enemy, secondMove)
	}
}
