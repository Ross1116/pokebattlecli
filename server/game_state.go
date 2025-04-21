package server

import (
	"github.com/ross1116/pokebattlecli/internal/battle"
)

type BattleState struct {
	Player1Username string
	Player2Username string

	Player1Team []*battle.BattlePokemon
	Player2Team []*battle.BattlePokemon

	Player1ActiveIndex int
	Player2ActiveIndex int

	TurnNumber   int
	Weather      string
	FieldEffects map[string]int

	LastTurnResults []string
}

func NewBattleState(p1Username, p2Username string, p1Team, p2Team []*battle.BattlePokemon) *BattleState {
	return &BattleState{
		Player1Username:    p1Username,
		Player2Username:    p2Username,
		Player1Team:        p1Team,
		Player2Team:        p2Team,
		Player1ActiveIndex: 0,
		Player2ActiveIndex: 0,
		TurnNumber:         1,
		FieldEffects:       make(map[string]int),
		LastTurnResults:    []string{},
	}
}

type PlayerAction struct {
	Type          string
	ActionIndex   int // 1 - 4 move, 0 switch
	SwitchToIndex int
}

type TurnResult struct {
	Description   []string
	DamageDealt   map[string]float64
	StatusChanges map[string]string
	Switches      map[string]int
}

func (b *BattleState) GetActivePokemons() (*battle.BattlePokemon, *battle.BattlePokemon) {
	return b.Player1Team[b.Player1ActiveIndex], b.Player2Team[b.Player2ActiveIndex]
}

func (b *BattleState) IsGameOver() bool {
	return battle.IsAllFainted(b.Player1Team) || battle.IsAllFainted(b.Player2Team)
}
