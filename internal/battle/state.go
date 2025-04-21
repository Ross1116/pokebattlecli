package battle

import (
	"fmt"
	"log"
	"time"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
	"github.com/ross1116/pokebattlecli/internal/stats"
)

type BattlePokemon struct {
	Base        *pokemon.Pokemon
	CurrentHP   float64
	MovePP      map[string]int
	Status      string
	StatusTurns int
	Fainted     bool
	StatStages  map[string]int
	Volatile    map[string]bool
	UniqueID    string
}

type PokemonSummary struct {
	Name      string
	HPPercent float64
	Status    string
	Fainted   bool
}

type PokemonFullView struct {
	Name       string
	Types      []string
	CurrentHP  float64
	MaxHP      float64
	Status     string
	StatStages map[string]int
	Volatile   map[string]bool
	Moves      []MoveView
}

type PokemonLimitedView struct {
	Name      string
	Types     []string
	HPPercent float64
	Status    string
	Volatile  map[string]bool
}

type MoveView struct {
	Name     string
	Type     string
	Power    int
	Accuracy int
	PP       int
	MaxPP    int
	Category string
}

func NewBattlePokemon(p *pokemon.Pokemon, moves []*pokemon.MoveInfo) *BattlePokemon {
	if p == nil {
		log.Println("Error: Attempted to create BattlePokemon from nil base Pokemon.")
		return nil
	}

	movePP := make(map[string]int)
	if moves != nil {
		for _, m := range moves {
			if m != nil {
				movePP[m.Name] = m.Pp
			}
		}
	}

	baseHp := 0
	if p.Stats != nil {
		for _, stat := range p.Stats {
			if stat.Stat.Name == "hp" {
				baseHp = stat.BaseStat
				break
			}
		}
	} else {
		log.Printf("Warning: Pokemon %s has nil Stats field.", p.Name)
	}
	maxCalculatedHP := stats.HpCalc(baseHp)

	statStages := make(map[string]int)

	return &BattlePokemon{
		Base:       p,
		CurrentHP:  maxCalculatedHP,
		MovePP:     movePP,
		Status:     "",
		Fainted:    false,
		StatStages: statStages,
		Volatile:   make(map[string]bool),
		UniqueID:   fmt.Sprintf("%s-%d", p.Name, time.Now().UnixNano()),
	}
}

func (bp *BattlePokemon) ApplyDamage(dmg float64) {
	if bp.Fainted {
		return
	}
	bp.CurrentHP -= dmg
	if bp.CurrentHP <= 0 {
		bp.CurrentHP = 0
		bp.Fainted = true
	}
}

func (bp *BattlePokemon) UseMove(moveName string) bool {
	if bp.Fainted {
		return false
	}
	if pp, ok := bp.MovePP[moveName]; ok && pp > 0 {
		bp.MovePP[moveName]--
		return true
	}
	return false
}

func (bp *BattlePokemon) ApplyStatus(newStatus string) {
	if bp.Status == "" && !bp.Fainted {
		bp.Status = newStatus
		bp.StatusTurns = 0
	}
}

func (bp *BattlePokemon) ApplyStatusWithDuration(status string, turns int) {
	if bp.Status == "" && !bp.Fainted {
		bp.Status = status
		bp.StatusTurns = turns
	}
}

func (bp *BattlePokemon) ApplyStatStage(stat string, change int) {
	currentStage := bp.StatStages[stat]
	newStage := currentStage + change
	if newStage > 6 {
		newStage = 6
	}
	if newStage < -6 {
		newStage = -6
	}
	bp.StatStages[stat] = newStage
}

func (bp *BattlePokemon) ApplyVolatileEffect(effect string) {
	if !bp.Fainted {
		bp.Volatile[effect] = true
	}
}

func (bp *BattlePokemon) RemoveVolatileEffect(effect string) {
	delete(bp.Volatile, effect)
}

func GetPokemonFullView(p *BattlePokemon) PokemonFullView {
	if p == nil || p.Base == nil {
		log.Println("Error: GetPokemonFullView called with nil BattlePokemon or Base.")
		return PokemonFullView{}
	}

	types := []string{}
	if p.Base.Types != nil {
		types = make([]string, len(p.Base.Types))
		for i, typeSlot := range p.Base.Types {
			types[i] = typeSlot.Type.Name
		}
	}

	maxHP := 0.0
	if p.Base.Stats != nil {
		for _, stat := range p.Base.Stats {
			if stat.Stat.Name == "hp" {
				maxHP = stats.HpCalc(stat.BaseStat)
				break
			}
		}
	}

	moveViews := []MoveView{}
	if p.Base.Moves != nil {
		moveViews = make([]MoveView, 0, len(p.Base.Moves))
		for _, moveSlot := range p.Base.Moves {
			moveName := moveSlot.Move.Name
			moveInfo, err := pokemon.FetchMoveByName(moveName)
			if err != nil || moveInfo == nil {
				log.Printf("Error fetching move details for %s: %v", moveName, err)
				continue
			}

			currentPP := 0
			if pp, ok := p.MovePP[moveName]; ok {
				currentPP = pp
			}

			moveViews = append(moveViews, MoveView{
				Name:     moveName,
				Type:     moveInfo.Type.Name,
				Power:    moveInfo.Power,
				Accuracy: moveInfo.Accuracy,
				PP:       currentPP,
				MaxPP:    moveInfo.Pp,
				Category: moveInfo.DamageClass.Name,
			})
		}
	}

	statStagesCopy := make(map[string]int, len(p.StatStages))
	for k, v := range p.StatStages {
		statStagesCopy[k] = v
	}
	volatileCopy := make(map[string]bool, len(p.Volatile))
	for k, v := range p.Volatile {
		volatileCopy[k] = v
	}

	return PokemonFullView{
		Name:       p.Base.Name,
		Types:      types,
		CurrentHP:  p.CurrentHP,
		MaxHP:      maxHP,
		Status:     p.Status,
		StatStages: statStagesCopy,
		Volatile:   volatileCopy,
		Moves:      moveViews,
	}
}

func GetPokemonLimitedView(p *BattlePokemon) PokemonLimitedView {
	if p == nil || p.Base == nil {
		log.Println("Error: GetPokemonLimitedView called with nil BattlePokemon or Base.")
		return PokemonLimitedView{}
	}

	types := []string{}
	if p.Base.Types != nil {
		types = make([]string, len(p.Base.Types))
		for i, typeSlot := range p.Base.Types {
			types[i] = typeSlot.Type.Name
		}
	}

	maxHP := 0.0
	if p.Base.Stats != nil {
		for _, stat := range p.Base.Stats {
			if stat.Stat.Name == "hp" {
				maxHP = stats.HpCalc(stat.BaseStat)
				break
			}
		}
	}

	hpPercent := 0.0
	if maxHP > 0 {
		hpPercent = (p.CurrentHP / maxHP) * 100.0
		if hpPercent < 0 {
			hpPercent = 0
		}
		if hpPercent > 100 {
			hpPercent = 100
		}
	}

	visibleVolatile := make(map[string]bool)
	if p.Volatile["confused"] {
		visibleVolatile["confused"] = true
	}

	return PokemonLimitedView{
		Name:      p.Base.Name,
		Types:     types,
		HPPercent: hpPercent,
		Status:    p.Status,
		Volatile:  visibleVolatile,
	}
}

func GetTeamSummary(team []*BattlePokemon) []PokemonSummary {
	if team == nil {
		return nil
	}
	summaries := make([]PokemonSummary, len(team))

	for i, pokemon := range team {
		if pokemon == nil || pokemon.Base == nil {
			summaries[i] = PokemonSummary{Name: "(Empty Slot)"}
			continue
		}

		maxHP := 0.0
		if pokemon.Base.Stats != nil {
			for _, stat := range pokemon.Base.Stats {
				if stat.Stat.Name == "hp" {
					maxHP = stats.HpCalc(stat.BaseStat)
					break
				}
			}
		}

		hpPercent := 0.0
		if maxHP > 0 {
			hpPercent = (pokemon.CurrentHP / maxHP) * 100.0
			if hpPercent < 0 {
				hpPercent = 0
			}
			if hpPercent > 100 {
				hpPercent = 100
			}
		}

		summaries[i] = PokemonSummary{
			Name:      pokemon.Base.Name,
			HPPercent: hpPercent,
			Status:    pokemon.Status,
			Fainted:   pokemon.Fainted,
		}
	}
	return summaries
}

func IsAllFainted(team []*BattlePokemon) bool {
	if team == nil || len(team) == 0 {
		return true
	}
	for _, p := range team {
		if p != nil && !p.Fainted {
			return false
		}
	}
	return true
}

// func (bp *BattlePokemon) HandleTurnEffects(weather string, fieldEffects map[string]int) []string {
// 	turnEvents := []string{}
// 	if bp.Fainted {
// 		return turnEvents
// 	}
//
// 	maxHP := 0.0
// 	hpStatFound := false
// 	if bp.Base != nil && bp.Base.Stats != nil {
// 		for _, stat := range bp.Base.Stats {
// 			if stat.Stat.Name == "hp" {
// 				maxHP = stats.HpCalc(stat.BaseStat)
// 				hpStatFound = true
// 				break
// 			}
// 		}
// 	}
// 	if !hpStatFound {
// 		log.Printf("Warning: Could not find Max HP for %s during HandleTurnEffects.", bp.Base.Name)
// 	}
//
// 	statusDamage := 0.0
// 	statusMsg := ""
// 	switch bp.Status {
// 	case "psn":
// 		if maxHP > 0 {
// 			statusDamage = maxHP / 8.0
// 		}
// 		statusMsg = fmt.Sprintf("%s took damage from poison!", bp.Base.Name)
// 	case "tox":
// 		if maxHP > 0 {
// 			statusDamage = maxHP / 8.0
// 		}
// 		statusMsg = fmt.Sprintf("%s took damage from poison!", bp.Base.Name)
// 	case "brn":
// 		if maxHP > 0 {
// 			statusDamage = maxHP / 16.0
// 		}
// 		statusMsg = fmt.Sprintf("%s took damage from its burn!", bp.Base.Name)
// 	}
// 	if statusDamage > 0 {
// 		bp.ApplyDamage(statusDamage)
// 		turnEvents = append(turnEvents, statusMsg)
// 		if bp.Fainted {
// 			turnEvents = append(turnEvents, fmt.Sprintf("%s fainted!", bp.Base.Name))
// 			return turnEvents
// 		}
// 	}
//
// 	if weather == "sandstorm" {
// 		isImmune := false
// 		if bp.Base != nil && bp.Base.Types != nil {
// 			for _, t := range bp.Base.Types {
// 				switch t.Type.Name {
// 				case "rock", "ground", "steel":
// 					isImmune = true
// 					break
// 				}
// 			}
// 		}
// 		if !isImmune && maxHP > 0 {
// 			weatherDmg := maxHP / 16.0
// 			bp.ApplyDamage(weatherDmg)
// 			turnEvents = append(turnEvents, fmt.Sprintf("%s took damage from the sandstorm!", bp.Base.Name))
// 			if bp.Fainted {
// 				turnEvents = append(turnEvents, fmt.Sprintf("%s fainted!", bp.Base.Name))
// 				return turnEvents
// 			}
// 		}
// 	}
//
// 	isGrounded := true
// 	if effectTurns, ok := fieldEffects["grassy-terrain"]; ok && effectTurns > 0 && isGrounded {
// 		if maxHP > 0 {
// 			healAmt := maxHP / 16.0
// 			bp.CurrentHP += healAmt
// 			if bp.CurrentHP > maxHP {
// 				bp.CurrentHP = maxHP
// 			}
// 			turnEvents = append(turnEvents, fmt.Sprintf("%s recovered some HP from the Grassy Terrain!", bp.Base.Name))
// 		}
// 	}
//
// 	if bp.Status == "slp" {
// 		if bp.StatusTurns > 0 {
// 			bp.StatusTurns--
// 			if bp.StatusTurns == 0 {
// 				bp.Status = ""
// 				turnEvents = append(turnEvents, fmt.Sprintf("%s woke up!", bp.Base.Name))
// 			} else {
// 			}
// 		} else {
// 			bp.Status = ""
// 			turnEvents = append(turnEvents, fmt.Sprintf("%s woke up! (Forced)", bp.Base.Name))
// 		}
// 	}
//
// 	return turnEvents
// }

