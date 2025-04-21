package battle

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

func DamageCalc(attacker *BattlePokemon, defender *BattlePokemon, move *pokemon.MoveInfo) (int, float64, []string) {
	events := []string{}
	if attacker == nil || defender == nil || move == nil || attacker.Base == nil || defender.Base == nil {
		log.Println("Error: DamageCalc received nil input.")
		return 0, 0, events
	}

	level := 100
	var atkStat, defStat int

	switch move.DamageClass.Name {
	case "physical":
		atkStat = int(stats.StatCalc(stats.GetStat(attacker.Base, "attack")))
		defStat = int(stats.StatCalc(stats.GetStat(defender.Base, "defense")))
		if attacker.Status == "brn" {
			atkStat /= 2
		}
	case "special":
		atkStat = int(stats.StatCalc(stats.GetStat(attacker.Base, "special-attack")))
		defStat = int(stats.StatCalc(stats.GetStat(defender.Base, "special-defense")))
	default:
		if move.Power > 0 {
			log.Printf("Unsupported move damage class: %s for move %s", move.DamageClass.Name, move.Name)
		}
		return 0, 0, events
	}
	if atkStat <= 0 || defStat <= 0 {
		log.Printf("Stat calculation error for move %s. Atk: %d, Def: %d", move.Name, atkStat, defStat)
		return 0, 0, events
	}
	if move.Power == 0 {
		return 0, 0, events
	}

	acc := float64(move.Accuracy)
	if acc > 0 {
		if rand.Float64()*100 > acc {
			events = append(events, fmt.Sprintf("%s's attack missed!", attacker.Base.Name))
			return 0, 0, events
		}
	}

	stab := 1.0
	if attacker.Base.Types != nil {
		for _, t := range attacker.Base.Types {
			if t.Type.Name == move.Type.Name {
				stab = 1.5
				break
			}
		}
	}

	effectiveness := 1.0
	if defender.Base.Types != nil && pokemon.TypeEffectiveness != nil {
		for _, t := range defender.Base.Types {
			if attackerEffectiveness, ok := pokemon.TypeEffectiveness[move.Type.Name]; ok {
				if mult, ok2 := attackerEffectiveness[t.Type.Name]; ok2 {
					effectiveness *= mult
				}
			}
		}
	}

	if effectiveness > 1.0 {
		events = append(events, "It's super effective!")
	} else if effectiveness < 1.0 && effectiveness > 0 {
		events = append(events, "It's not very effective...")
	} else if effectiveness == 0 {
		events = append(events, fmt.Sprintf("It doesn't affect %s!", defender.Base.Name))
		return 0, 0, events
	}

	baseDmg := (((2.0 * float64(level) / 5.0) + 2.0) * float64(move.Power) * float64(atkStat) / float64(defStat)) / 50.0
	if baseDmg < 1.0 && effectiveness > 0 {
		baseDmg = 1.0
	} else {
		baseDmg += 2.0
	}

	randomFactor := 0.85 + (rand.Float64() * 0.15)

	critChance := 6.25
	critRoll := rand.Float64() * 100
	critMultiplier := 1.0
	if critRoll < critChance {
		events = append(events, "Critical hit!")
		critMultiplier = 1.5
	}

	finalDmg := baseDmg * stab * effectiveness * randomFactor * critMultiplier

	roundedDmg := int(math.Floor(finalDmg))
	if roundedDmg < 1 && effectiveness > 0 {
		roundedDmg = 1
	}
	percent := 0.0
	baseHp := stats.GetStat(defender.Base, "hp")
	if baseHp > 0 {
		totalHp := stats.HpCalc(baseHp)
		if totalHp > 0 {
			percent = (float64(roundedDmg) / totalHp) * 100.0
		}
	}

	return roundedDmg, percent, events
}

func ExecuteBattleTurn(player *BattlePokemon, enemy *BattlePokemon, playerMove *pokemon.MoveInfo, enemyMove *pokemon.MoveInfo) []string {
	turnEvents := []string{}
	first, second, firstMove, secondMove := ResolveTurn(player, enemy, playerMove, enemyMove)

	if first != nil && firstMove != nil {
		var moveEvents []string
		if first == player {
			moveEvents = ProcessPlayerTurn(first, second, firstMove)
		} else {
			moveEvents = ProcessEnemyTurn(second, first, firstMove)
		}
		turnEvents = append(turnEvents, moveEvents...)
		if second.Fainted {
			goto EndTurnEffects
		}
	} else if first != nil && !first.Fainted {
	}

	if second != nil && secondMove != nil && !second.Fainted {
		var moveEvents []string
		if second == player {
			moveEvents = ProcessPlayerTurn(second, first, secondMove)
		} else {
			moveEvents = ProcessEnemyTurn(first, second, secondMove)
		}
		turnEvents = append(turnEvents, moveEvents...)
	} else if second != nil && !second.Fainted {
	}

EndTurnEffects:
	if first != nil && !first.Fainted {
		effectEvents := first.HandleTurnEffects()
		turnEvents = append(turnEvents, effectEvents...)
	}
	if second != nil && !second.Fainted {
		if first == nil || !first.Fainted {
			effectEvents := second.HandleTurnEffects()
			turnEvents = append(turnEvents, effectEvents...)
		}
	}

	return turnEvents
}

