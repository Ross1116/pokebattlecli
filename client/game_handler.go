package client

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var clear map[string]func()

func init() {
	clear = make(map[string]func())
	clear["linux"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["darwin"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func CallClear() {
	value, ok := clear[runtime.GOOS]
	if ok {
		value()
	} else {
		fmt.Print(strings.Repeat("\n", 50))
		log.Println("Warning: Unsupported platform for screen clear. Using newline fallback.")
	}
}

func (c *Client) startGameMode() {
	c.GameActive = true
	log.Println("Entered game mode.")
}

func (c *Client) endGameMode() {
	c.GameActive = false
	c.InMatch = false
	c.Opponent = ""
	c.PlayerSquad = nil
	c.EnemySquad = nil
	c.PlayerMaxHPs = nil
	c.EnemyMaxHPs = nil
	c.LastTurnDescription = nil
	fmt.Println("\n=== Exited Battle Mode ===")
	fmt.Print("> ")
	log.Println("Exited game mode.")
}

func (c *Client) handleTurnRequest(msg Message) {
	if !c.GameActive {
		log.Println("Warning: Received turn_request while not in game mode. Entering game mode.")
		c.startGameMode()
	}

	turnNumberF, _ := msg.Message["turn"].(float64)
	turnNumber := int(turnNumberF)
	availableMovesInterface, _ := msg.Message["available_moves"].([]interface{})
	availableMoves := make([]string, len(availableMovesInterface))
	for i, move := range availableMovesInterface {
		availableMoves[i], _ = move.(string)
	}

	fmt.Printf("\n=== TURN %d === vs %s\n", turnNumber, c.Opponent)

	fmt.Println("\nYour Squad:")
	if c.PlayerSquad == nil || len(c.PlayerSquad) == 0 || c.PlayerMaxHPs == nil {
		fmt.Println("(Squad information not available or not fully initialized)")
	} else {
		for i, poke := range c.PlayerSquad {
			if poke == nil || poke.Base == nil || i >= len(c.PlayerMaxHPs) {
				fmt.Printf("%d. (Error loading Pokemon data)\n", i+1)
				continue
			}

			maxHP := c.PlayerMaxHPs[i]
			hpPercent := 0.0
			if maxHP > 0 {
				hpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
			}

			status := ""
			if poke.Fainted {
				status = "[FNT]"
			} else if poke.Status != "" {
				status = fmt.Sprintf("[%s]", strings.ToUpper(poke.Status))
			}

			activeIndicator := "  "
			if i == c.PlayerActiveIdx {
				activeIndicator = "->"
			}

			fmt.Printf("%s %d. %-12s HP: %3.0f%% %s\n",
				activeIndicator,
				i+1,
				poke.Base.Name,
				hpPercent,
				status,
			)
		}
	}
	fmt.Println("-------------------------")

	if len(availableMoves) > 0 {
		fmt.Println("Available moves:")
		for i, move := range availableMoves {
			fmt.Printf("%d. %s\n", i+1, move)
		}
	} else {
		fmt.Println("No moves available! (You might need to switch)")
	}
	fmt.Println("-------------------------")
	fmt.Print("\nEnter your action (move <number> or switch <number>): ")
}

func (c *Client) handleTurnResult(msg Message) {

	c.applyBattleStateUpdate(msg)

	fmt.Println("\n=== TURN RESULT ===")
	if len(c.LastTurnDescription) == 0 {
		fmt.Println("(No specific events reported)")
	} else {
		for _, descStr := range c.LastTurnDescription {
			fmt.Println(descStr)
		}
	}

	yourHpPercent := 0.0
	oppHpPercent := 0.0

	if c.PlayerSquad != nil && c.PlayerActiveIdx >= 0 && c.PlayerActiveIdx < len(c.PlayerSquad) && c.PlayerSquad[c.PlayerActiveIdx] != nil && c.PlayerActiveIdx < len(c.PlayerMaxHPs) {
		poke := c.PlayerSquad[c.PlayerActiveIdx]
		maxHP := c.PlayerMaxHPs[c.PlayerActiveIdx]
		if maxHP > 0 {
			yourHpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
		}
	}

	if c.EnemySquad != nil && c.EnemyActiveIdx >= 0 && c.EnemyActiveIdx < len(c.EnemySquad) && c.EnemySquad[c.EnemyActiveIdx] != nil && c.EnemyActiveIdx < len(c.EnemyMaxHPs) {
		poke := c.EnemySquad[c.EnemyActiveIdx]
		maxHP := c.EnemyMaxHPs[c.EnemyActiveIdx]
		if maxHP > 0 {
			oppHpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
		}
	}

	yourHpStr := fmt.Sprintf("Your Pokemon HP: %.1f%%", yourHpPercent)
	oppHpStr := fmt.Sprintf("Opponent HP: %.1f%%", oppHpPercent)
	fmt.Printf("\n--- %-30s | %-30s ---\n", yourHpStr, oppHpStr)

	fmt.Println("===================")
}

func (c *Client) handleOpponentDisconnected(msg Message) {
	opponentName, _ := msg.Message["opponent"].(string)
	if opponentName == "" {
		opponentName = "Opponent"
	}
	fmt.Printf("\n!!! %s has disconnected from the match! !!!\n", opponentName)
	c.endGameMode()
}

func (c *Client) handleGameInput(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		fmt.Print("Please enter an action: ")
		return
	}
	command := strings.ToLower(parts[0])

	if len(parts) == 1 {
		if moveNum, err := strconv.Atoi(parts[0]); err == nil {
			if moveNum < 1 || moveNum > 4 {
				fmt.Println("Move number must be between 1 and 4.")
				fmt.Print("Enter your action: ")
				return
			}
			c.sendAction("move", moveNum, 0)
			return
		}
	}

	if len(parts) < 2 {
		fmt.Println("Invalid command format. Use 'move <number>' or 'switch <number>'.")
		fmt.Print("Enter your action: ")
		return
	}
	actionIndex, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("Invalid number format for index.")
		fmt.Print("Enter your action: ")
		return
	}

	switch command {
	case "move", "m":
		if actionIndex < 1 || actionIndex > 4 {
			fmt.Println("Move index must be between 1 and 4.")
			fmt.Print("Enter your action: ")
			return
		}
		c.sendAction("move", actionIndex, 0)
	case "switch", "s":
		if actionIndex < 1 || actionIndex > 6 {
			fmt.Println("Switch index must be between 1 and 6.")
			fmt.Print("Enter your action: ")
			return
		}
		c.sendAction("switch", 0, actionIndex-1)
	default:
		fmt.Println("Unknown action. Use 'move <number>' or 'switch <number>'.")
		fmt.Print("Enter your action: ")
	}
}

func (c *Client) sendAction(actionType string, moveIndex, switchIndex int) {
	actionStr := fmt.Sprintf("GAME_ACTION_MARKER|%s|%d|%d", actionType, moveIndex, switchIndex)
	log.Printf("Sending action: %s", actionStr)

	if c.Conn == nil || !c.Connected {
		log.Println("Error: Cannot send action, not connected.")
		fmt.Println("Error: Connection lost. Cannot send action.")
		c.Disconnect()
		return
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Conn.Write([]byte(actionStr))
	c.Conn.SetWriteDeadline(time.Time{})

	if err != nil {
		log.Printf("Failed to send action: %v", err)
		fmt.Println("Error sending action to server. Disconnecting.")
		c.Disconnect()
	} else {
		fmt.Printf("Sent %s action to server...\n", actionType)
	}
}

