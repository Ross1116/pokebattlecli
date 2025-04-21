package client

import (
	"fmt"
	"log"
)

func (c *Client) processPlayerList(msg Message) {
	playersInterface, ok := msg.Message["players"]
	if !ok {
		log.Println("No players field in player_list message")
		return
	}

	players, ok := playersInterface.([]interface{})
	if !ok {
		log.Printf("Invalid players format: %T", playersInterface)
		return
	}

	fmt.Println("Connected players:")
	for i, player := range players {
		playerName, ok := player.(string)
		if !ok {
			continue
		}
		fmt.Printf("%d. %s", i+1, playerName)
		if playerName == c.Config.Username {
			fmt.Print(" (you)")
		}
		fmt.Println()
	}
}

func (c *Client) processMatchStart(msg Message) {
	opponentInterface, ok := msg.Message["opponent"]
	if !ok {
		log.Println("No opponent field in match_start message")
		return
	}

	opponent, ok := opponentInterface.(string)
	if !ok {
		log.Printf("Invalid opponent format: %T", opponentInterface)
		return
	}

	c.Opponent = opponent
	c.InMatch = true
	fmt.Printf("Match started with %s!\n", opponent)
}

func (c *Client) processGameStart(msg Message) {
	yourSquadInterface, ok := msg.Message["your_squad"]
	if !ok {
		log.Println("No your_squad field in game_start message")
		return
	}

	opponentSquadInterface, ok := msg.Message["opponent_squad"]
	if !ok {
		log.Println("No opponent_squad field in game_start message")
		return
	}

	yourSquadRaw, ok := yourSquadInterface.([]interface{})
	if !ok {
		log.Printf("Invalid your_squad format: %T", yourSquadInterface)
		return
	}

	opponentSquadRaw, ok := opponentSquadInterface.([]interface{})
	if !ok {
		log.Printf("Invalid opponent_squad format: %T", opponentSquadInterface)
		return
	}

	yourSquad := make([]string, len(yourSquadRaw))
	for i, v := range yourSquadRaw {
		yourSquad[i], ok = v.(string)
		if !ok {
			log.Printf("Non-string pokemon in your_squad: %T", v)
			return
		}
	}

	opponentSquad := make([]string, len(opponentSquadRaw))
	for i, v := range opponentSquadRaw {
		opponentSquad[i], ok = v.(string)
		if !ok {
			log.Printf("Non-string pokemon in opponent_squad: %T", v)
			return
		}
	}

	c.InMatch = true

	fmt.Printf("Match started against %s!\n", c.Opponent)
	fmt.Println("\nYour squad:")
	for _, pokemon := range yourSquad {
		fmt.Printf("- %s\n", pokemon)
	}

	fmt.Println("\nOpponent's squad:")
	for _, pokemon := range opponentSquad {
		fmt.Printf("- %s\n", pokemon)
	}

	c.setupBattleState(yourSquad, opponentSquad)

	c.startGameMode()
}

func (c *Client) processGameEnd(msg Message) {
	resultInterface, ok := msg.Message["result"]
	if !ok {
		log.Println("No result field in game_end message")
		return
	}

	result, ok := resultInterface.(string)
	if !ok {
		log.Printf("Invalid result format: %T", resultInterface)
		return
	}

	if c.GameActive {
		c.endGameMode()
	}

	c.InMatch = false
	opponent := c.Opponent
	c.Opponent = ""

	switch result {
	case "win":
		fmt.Printf("You won the match against %s!\n", opponent)
	case "lose":
		fmt.Printf("You lost the match against %s.\n", opponent)
	default:
		fmt.Printf("Match with %s ended: %s\n", opponent, result)
	}
}

func (c *Client) processGameUpdate(msg Message) {

	if messages, ok := msg.Message["battle_messages"].([]interface{}); ok {
		for _, msgInterface := range messages {
			if battleMsg, ok := msgInterface.(string); ok {
				fmt.Println(battleMsg)
			}
		}
	}
}

// Add this method to set up the battle state
func (c *Client) setupBattleState(yourSquad, opponentSquad []string) {
	// This would initialize your battle state with the squads received from the server
	// For now, this is a stub - you would need to implement this with your battle system
	fmt.Println("\nSetting up battle with received squads...")

	// You would replace this with actual initialization from your battle system:
	// c.PlayerSquad, c.EnemySquad, c.PlayerMovesets, c.EnemyMovesets, c.PlayerActiveIdx, c.EnemyActiveIdx =
	//     battle.SetupSpecificSquads(yourSquad, opponentSquad)

	// This should also initialize:
	// c.PlayerMaxHPs and c.EnemyMaxHPs
}
