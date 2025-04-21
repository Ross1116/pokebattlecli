package battle

import (
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
	movePP := make(map[string]int)
	for _, m := range moves {
		movePP[m.Name] = m.Pp
	}

	baseHp := 0
	for _, stat := range p.Stats {
		if stat.Stat.Name == "hp" {
			baseHp = stat.BaseStat
			break
		}
	}

	return &BattlePokemon{
		Base:       p,
		CurrentHP:  stats.HpCalc(baseHp),
		MovePP:     movePP,
		Status:     "",
		Fainted:    false,
		StatStages: make(map[string]int),
		Volatile:   make(map[string]bool),
	}
}

func (bp *BattlePokemon) ApplyDamage(dmg float64) {
	bp.CurrentHP -= dmg
	if bp.CurrentHP <= 0 {
		bp.CurrentHP = 0
		bp.Fainted = true
	}
}

func (bp *BattlePokemon) UseMove(moveName string) bool {
	if pp, ok := bp.MovePP[moveName]; ok && pp > 0 {
		bp.MovePP[moveName]--
		return true
	}
	return false
}

func (bp *BattlePokemon) ApplyStatus(newStatus string) {
	bp.Status = newStatus
}

func (bp *BattlePokemon) ApplyStatusWithDuration(status string, turns int) {
	bp.Status = status
	bp.StatusTurns = turns
}

func (bp *BattlePokemon) ApplyStatStage(stat string, stage int) {
	bp.StatStages[stat] += stage
}

func (bp *BattlePokemon) ApplyVolatileEffect(effect string) {
	bp.Volatile[effect] = true
}

func (bp *BattlePokemon) RemoveVolatileEffect(effect string) {
	delete(bp.Volatile, effect)
}

func GetPokemonFullView(p *BattlePokemon) PokemonFullView {
	types := make([]string, len(p.Base.Types))
	for i, typeSlot := range p.Base.Types {
		types[i] = typeSlot.Type.Name
	}

	maxHP := 0.0
	for _, stat := range p.Base.Stats {
		if stat.Stat.Name == "hp" {
			maxHP = stats.HpCalc(stat.BaseStat)
			break
		}
	}

	moveViews := make([]MoveView, 0)

	for _, moveSlot := range p.Base.Moves {
		moveName := moveSlot.Move.Name
		moveInfo, _ := pokemon.FetchMoveByName(moveName)

		moveViews = append(moveViews, MoveView{
			Name:     moveName,
			Type:     moveInfo.Type.Name,
			Power:    moveInfo.Power,
			Accuracy: moveInfo.Accuracy,
			PP:       p.MovePP[moveName],
			MaxPP:    moveInfo.Pp,
			Category: moveInfo.DamageClass.Name,
		})
	}

	return PokemonFullView{
		Name:       p.Base.Name,
		Types:      types,
		CurrentHP:  p.CurrentHP,
		MaxHP:      maxHP,
		Status:     p.Status,
		StatStages: p.StatStages,
		Volatile:   p.Volatile,
		Moves:      moveViews,
	}
}

func GetPokemonLimitedView(p *BattlePokemon) PokemonLimitedView {
	types := make([]string, len(p.Base.Types))
	for i, typeSlot := range p.Base.Types {
		types[i] = typeSlot.Type.Name
	}

	maxHP := 0.0
	for _, stat := range p.Base.Stats {
		if stat.Stat.Name == "hp" {
			maxHP = stats.HpCalc(stat.BaseStat)
			break
		}
	}

	hpPercent := 0.0
	if maxHP > 0 {
		hpPercent = (p.CurrentHP / maxHP) * 100.0
	}

	return PokemonLimitedView{
		Name:      p.Base.Name,
		Types:     types,
		HPPercent: hpPercent,
		Status:    p.Status,
		Volatile:  p.Volatile,
	}
}

func GetTeamSummary(team []*BattlePokemon) []PokemonSummary {
	summaries := make([]PokemonSummary, len(team))

	for i, pokemon := range team {
		maxHP := 0.0
		for _, stat := range pokemon.Base.Stats {
			if stat.Stat.Name == "hp" {
				maxHP = stats.HpCalc(stat.BaseStat)
				break
			}
		}

		hpPercent := 0.0
		if maxHP > 0 {
			hpPercent = (pokemon.CurrentHP / maxHP) * 100.0
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
	for _, p := range team {
		if !p.Fainted {
			return false
		}
	}
	return true
}
