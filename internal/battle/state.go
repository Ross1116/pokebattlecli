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

