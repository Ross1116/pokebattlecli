package battle_test

import (
	"testing"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

func TestExecuteBattleTurn(t *testing.T) {
	charmander, err := pokemon.FetchPokemonData("charmander")
	if err != nil {
		t.Fatalf("Failed to fetch Charmander: %v", err)
	}

	squirtle, err := pokemon.FetchPokemonData("squirtle")
	if err != nil {
		t.Fatalf("Failed to fetch Squirtle: %v", err)
	}

	ember, err := pokemon.FetchMoveByName("ember")
	if err != nil {
		t.Fatalf("Failed to fetch move Ember: %v", err)
	}

	tackle, err := pokemon.FetchMoveByName("tackle")
	if err != nil {
		t.Fatalf("Failed to fetch move Tackle: %v", err)
	}

	charmanderBP := &battle.BattlePokemon{
		Base:      charmander,
		CurrentHP: 100,
		MovePP:    map[string]int{"ember": 10},
	}
	squirtleBP := &battle.BattlePokemon{
		Base:      squirtle,
		CurrentHP: 100,
		MovePP:    map[string]int{"tackle": 10},
	}

	battle.ExecuteBattleTurn(charmanderBP, squirtleBP, ember, tackle)

	if charmanderBP.Fainted {
		t.Logf("Charmander fainted.")
	}

	if squirtleBP.Fainted {
		t.Logf("Squirtle fainted.")
	}

	if charmanderBP.CurrentHP <= 0 || squirtleBP.CurrentHP <= 0 {
		t.Logf("Current HP: Charmander %f | Squirtle %f", charmanderBP.CurrentHP, squirtleBP.CurrentHP)
	}
}

func TestPriorityOverridesSpeed(t *testing.T) {
	sneasel, err := pokemon.FetchPokemonData("sneasel") // faster
	if err != nil {
		t.Fatalf("Failed to fetch Sneasel: %v", err)
	}

	slowbro, err := pokemon.FetchPokemonData("slowbro") //slow
	if err != nil {
		t.Fatalf("Failed to fetch Slowbro: %v", err)
	}

	tackle, err := pokemon.FetchMoveByName("tackle") // priority 0
	if err != nil {
		t.Fatalf("Failed to fetch Tackle: %v", err)
	}

	quickAttack, err := pokemon.FetchMoveByName("quick-attack") // priority +1
	if err != nil {
		t.Fatalf("Failed to fetch Quick Attack: %v", err)
	}

	sneaselBP := &battle.BattlePokemon{
		Base:      sneasel,
		CurrentHP: 100,
		MovePP:    map[string]int{"tackle": 10},
	}
	slowbroBP := &battle.BattlePokemon{
		Base:      slowbro,
		CurrentHP: 100,
		MovePP:    map[string]int{"quick-attack": 10},
	}

	battle.ExecuteBattleTurn(slowbroBP, sneaselBP, quickAttack, tackle)

	t.Logf("Post-turn HP: Slowbro: %.1f | Sneasel: %.1f", slowbroBP.CurrentHP, sneaselBP.CurrentHP)
}
