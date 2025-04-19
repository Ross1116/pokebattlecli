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
