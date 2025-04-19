package battle

import (
	"fmt"
	"math/rand"

	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

// Helper function for enemy attacks
func EnemyAttack(attacker, defender *BattlePokemon, moveSet []*pokemon.MoveInfo) {
	opponentMoveIndex := rand.Intn(len(moveSet))
	opponentMoveData := moveSet[opponentMoveIndex]

	if attacker.MovePP[opponentMoveData.Name] > 0 {
		attacker.UseMove(opponentMoveData.Name)
		opponentDamage, opponentPercent := DamageCalc(attacker.Base, defender.Base, opponentMoveData)
		defender.ApplyDamage(float64(opponentDamage))

		fmt.Printf("%s used %s! It dealt %d damage! (~%.2f%% of your Pokémon's HP)\n",
			attacker.Base.Name, opponentMoveData.Name, opponentDamage, opponentPercent)
	} else {
		fmt.Printf("%s tried to use %s but has no PP left!\n",
			attacker.Base.Name, opponentMoveData.Name)
	}
}

// Display battle status helper
func DisplayBattleStatus(player, enemy *BattlePokemon, playerMaxHP, enemyMaxHP float64) {
	playerHPPercent := (player.CurrentHP / playerMaxHP) * 100
	enemyHPPercent := (enemy.CurrentHP / enemyMaxHP) * 100

	fmt.Printf("\nYour %s's HP: %.2f/%.2f (%.2f%%)",
		player.Base.Name, player.CurrentHP, playerMaxHP, playerHPPercent)
	if player.Status != "" {
		fmt.Printf(" [%s]", player.Status)
	}
	fmt.Println()

	fmt.Printf("Enemy %s's HP: %.2f/%.2f (%.2f%%)",
		enemy.Base.Name, enemy.CurrentHP, enemyMaxHP, enemyHPPercent)
	if enemy.Status != "" {
		fmt.Printf(" [%s]", enemy.Status)
	}
	fmt.Println()
}
