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

	"github.com/ross1116/pokebattlecli/internal/stats"
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
}

func (c *Client) endGameMode() {
	c.GameActive = false
	c.InMatch = false
	c.Opponent = ""
	fmt.Println("=== Exited Battle Mode ===")
}

func (c *Client) handleTurnRequest(msg Message) {
	if !c.GameActive {
		log.Println("Warning: Received turn_request while not in game mode.")
		c.startGameMode()
	}

	turnNumberF, _ := msg.Message["turn"].(float64)
	turnNumber := int(turnNumberF)
	availableMovesInterface, _ := msg.Message["available_moves"].([]interface{})
	availableMoves := make([]string, len(availableMovesInterface))
	for i, move := range availableMovesInterface {
		availableMoves[i], _ = move.(string)
	}

	// CallClear()
	fmt.Printf("\n=== TURN %d === vs %s\n", turnNumber, c.Opponent)

	fmt.Println("\nYour Squad:")
	if c.PlayerSquad == nil || len(c.PlayerSquad) == 0 {
		fmt.Println("(Squad information not available - setupBattleState needs implementation)")
	} else {
		for i, poke := range c.PlayerSquad {
			if poke == nil || poke.Base == nil {
				fmt.Printf("%d. (Error loading Pokemon)\n", i+1)
				continue
			}
			maxHP := 0.0
			hpBaseStat := stats.GetStat(poke.Base, "hp")
			if hpBaseStat > 0 {
				maxHP = stats.HpCalc(hpBaseStat)
			}

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
		fmt.Println("No moves available!")
	}
	fmt.Println("-------------------------")
	fmt.Print("\nEnter your action (move <number> or switch <number>): ")
}

func (c *Client) handleTurnResult(msg Message) {
	// CallClear()

	descriptionInterface, okD := msg.Message["description"].([]interface{})
	yourHpPercent, okY := msg.Message["your_hp_percent"].(float64)
	oppHpPercent, okO := msg.Message["opponent_hp_percent"].(float64)

	if !okD {
		log.Println("Invalid 'description' format in turn_result message")
		return
	}
	if !okY || !okO {
		log.Println("Warning: HP percentages missing or invalid in turn_result message")
	}

	fmt.Println("\n=== TURN RESULT ===")
	if len(descriptionInterface) == 0 {
		fmt.Println("(No specific events reported)")
	} else {
		for _, desc := range descriptionInterface {
			if descStr, ok := desc.(string); ok {
				fmt.Println(descStr)
			}
		}
	}
	if okY && okO {
		yourHpStr := fmt.Sprintf("Your Pokemon HP: %.1f%%", yourHpPercent)
		oppHpStr := fmt.Sprintf("Opponent HP: %.1f%%", oppHpPercent)
		fmt.Printf("\n--- %-30s | %-30s ---\n", yourHpStr, oppHpStr)
	} else {
		fmt.Printf("\n--- (HP data unavailable) ---\n")
	}
	fmt.Println("===================")
}

func (c *Client) handleOpponentDisconnected(msg Message) {
	opponentName, _ := msg.Message["opponent"].(string)
	if opponentName == "" {
		opponentName = "Opponent"
	}
	// CallClear()
	fmt.Printf("\n!!! %s has disconnected from the match! !!!\n", opponentName)
	c.endGameMode()
	fmt.Print("> ")
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
		return
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Conn.Write([]byte(actionStr))
	c.Conn.SetWriteDeadline(time.Time{})

	if err != nil {
		log.Printf("Failed to send action: %v", err)
		c.Disconnect()
	} else {
		fmt.Printf("Sent %s action to server...\n", actionType)
	}
}
