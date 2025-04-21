package client

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

func (c *Client) startGameMode() {
	c.GameActive = true
	fmt.Println("\n=== BATTLE MODE ===")
	fmt.Println("Type 'move <number>' to use a move or 'switch <number>' to switch Pokemon.")
}

func (c *Client) endGameMode() {
	c.GameActive = false
}

func (c *Client) handleTurnRequest(msg Message) {
	turnNumber, _ := msg.Message["turn"].(float64)
	availableMovesInterface, _ := msg.Message["available_moves"].([]interface{})

	availableMoves := make([]string, len(availableMovesInterface))
	for i, move := range availableMovesInterface {
		availableMoves[i], _ = move.(string)
	}

	fmt.Printf("\n=== TURN %d ===\n", int(turnNumber))
	fmt.Println("Available moves:")
	for i, move := range availableMoves {
		fmt.Printf("%d. %s\n", i+1, move)
	}
	fmt.Println("\nEnter your action (move <number> or switch <number>):")
}

func (c *Client) handleTurnResult(msg Message) {
	descriptionInterface, _ := msg.Message["description"].([]interface{})

	fmt.Println("\n=== TURN RESULT ===")
	for _, desc := range descriptionInterface {
		if descStr, ok := desc.(string); ok {
			fmt.Println(descStr)
		}
	}
	fmt.Println()
}

func (c *Client) handleOpponentDisconnected(msg Message) {
	opponentInterface, _ := msg.Message["opponent"].(string)
	fmt.Printf("\nYour opponent %s has disconnected from the match.\n", opponentInterface)
	c.endGameMode()
	c.InMatch = false
	c.Opponent = ""
}

func (c *Client) handleGameInput(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	// Handle just a number as input (shortcut for move)
	if len(parts) == 1 {
		if moveNum, err := strconv.Atoi(parts[0]); err == nil {
			if moveNum < 1 || moveNum > 4 {
				fmt.Println("Move number must be between 1 and 4")
				return
			}
			c.sendAction("move", moveNum, 0)
			return
		}
	}

	if len(parts) < 2 {
		fmt.Println("Invalid command. Use 'move <number>' or 'switch <number>'")
		return
	}

	actionType := parts[0]
	actionIndex, err := strconv.Atoi(parts[1])

	if err != nil {
		fmt.Println("Invalid number format")
		return
	}

	switch actionType {
	case "move":
		if actionIndex < 1 || actionIndex > 4 {
			fmt.Println("Move index must be between 1 and 4")
			return
		}
		c.sendAction("move", actionIndex, 0)
	case "switch":
		// Validate switch index based on your squad size
		if actionIndex < 1 || actionIndex > 6 {
			fmt.Println("Switch index must be between 1 and 6")
			return
		}
		c.sendAction("switch", 0, actionIndex-1)
	default:
		fmt.Println("Unknown action. Use 'move <number>' or 'switch <number>'")
	}
}

func (c *Client) sendAction(actionType string, moveIndex, switchIndex int) {
	// Use a simple format that's easy to parse
	actionStr := fmt.Sprintf("GAME_ACTION_MARKER|%s|%d|%d",
		actionType, moveIndex, switchIndex)

	log.Printf("Sending action: %s", actionStr)

	_, err := c.Conn.Write([]byte(actionStr))
	if err != nil {
		log.Printf("Failed to send action: %v", err)
		c.Connected = false
	} else {
		fmt.Printf("Sent %s action to server...\n", actionType)
	}
}
