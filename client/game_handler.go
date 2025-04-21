package client

import (
	"encoding/json"
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
	clear["linux"] = func() { cmd := exec.Command("clear"); cmd.Stdout = os.Stdout; cmd.Run() }
	clear["windows"] = func() { cmd := exec.Command("cmd", "/c", "cls"); cmd.Stdout = os.Stdout; cmd.Run() }
	clear["darwin"] = func() { cmd := exec.Command("clear"); cmd.Stdout = os.Stdout; cmd.Run() }
}
func CallClear() {
	value, ok := clear[runtime.GOOS]
	if ok {
		value()
	} else {
		fmt.Print(strings.Repeat("\n", 50))
		log.Println("Warning: Unsupported platform for screen clear.")
	}
}

func (c *Client) startGameMode() {
	c.GameActive = true
	c.AwaitingForcedSwitch = false
	log.Println("Entered game mode.")
}

func (c *Client) endGameMode() {
	c.GameActive = false
	c.AwaitingForcedSwitch = false
	c.InMatch = false
	c.Opponent = ""
	c.PlayerSquad = nil
	c.EnemySquad = nil
	c.PlayerMaxHPs = nil
	c.EnemyMaxHPs = nil
	c.LastTurnDescription = nil
	c.LastAvailableMovesInfo = nil
	fmt.Println("\n=== Exited Battle Mode ===")
	log.Println("Exited game mode.")
}

func (c *Client) handleTurnRequest(msg Message) {
	if !c.GameActive {
		log.Println("Warning: Received turn_request while not in game mode.")
		return
	}
	c.AwaitingForcedSwitch = false
	turnNumberF, _ := msg.Message["turn"].(float64)
	turnNumber := int(turnNumberF)
	forceSwitch, _ := msg.Message["force_switch"].(bool)
	c.LastAvailableMovesInfo = nil
	if movesInfoInterface, ok := msg.Message["available_moves_info"]; ok {
		jsonBytes, err := json.Marshal(movesInfoInterface)
		if err == nil {
			var movesInfo []MoveStateInfo
			err = json.Unmarshal(jsonBytes, &movesInfo)
			if err == nil {
				c.LastAvailableMovesInfo = movesInfo
			} else {
				log.Printf("Error unmarshaling available_moves_info: %v", err)
			}
		} else {
			log.Printf("Error marshaling available_moves_info for unmarshal: %v", err)
		}
	} else {
		log.Println("Warning: 'available_moves_info' not found in turn_request. PP info unavailable.")
	}

	fmt.Printf("\n=== TURN %d === vs %s\n", turnNumber, c.Opponent)
	fmt.Println("\nYour Squad:")
	if c.PlayerSquad == nil || len(c.PlayerSquad) == 0 || c.PlayerMaxHPs == nil {
		fmt.Println("(Squad information not available)")
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
			fmt.Printf("%s %d. %-12s HP: %3.0f%% %s\n", activeIndicator, i+1, poke.Base.Name, hpPercent, status)
		}
	}
	fmt.Println("-------------------------")
	if forceSwitch {
		fmt.Println("Your active Pokemon has fainted! You must switch.")
		fmt.Println("Available Pokemon to switch to:")
		switchCount := 0
		if c.PlayerSquad != nil && len(c.PlayerMaxHPs) == len(c.PlayerSquad) {
			for i, poke := range c.PlayerSquad {
				if poke != nil && !poke.Fainted && i != c.PlayerActiveIdx {
					maxHP := c.PlayerMaxHPs[i]
					hpPercent := 0.0
					if maxHP > 0 {
						hpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
					}
					fmt.Printf("%d. %s (%.0f%% HP)\n", i+1, poke.Base.Name, hpPercent)
					switchCount++
				}
			}
		}
		if switchCount == 0 {
			fmt.Println("!!! No available Pokemon to switch to!")
		}
		fmt.Println("-------------------------")
		fmt.Print("\nEnter action (switch <number>): ")
	} else {
		fmt.Println("Available moves:")
		if c.LastAvailableMovesInfo != nil && len(c.LastAvailableMovesInfo) > 0 {
			for i, moveInfo := range c.LastAvailableMovesInfo {
				ppIndicator := ""
				if moveInfo.CurrentPP <= 0 {
					ppIndicator = " (NO PP)"
				}
				fmt.Printf("%d. %-15s (%d/%d PP)%s\n", i+1, moveInfo.Name, moveInfo.CurrentPP, moveInfo.MaxPP, ppIndicator)
			}
		} else {
			fmt.Println("No moves available!")
		}
		fmt.Println("-------------------------")
		fmt.Print("\nEnter your action (move <number> or switch <number>): ")
	}
}

func (c *Client) handleTurnResult(msg Message) {
	CallClear()
	c.applyBattleStateUpdate(msg)

	fmt.Println("\n=== TURN RESULT ===")
	if len(c.LastTurnDescription) == 0 {
		fmt.Println("(No specific events reported)")
	} else {
		for _, descStr := range c.LastTurnDescription {
			fmt.Println(descStr)
		}
	}

	yourHpPercent, oppHpPercent := 0.0, 0.0
	yourPokemonName := "(Your Pokemon)"
	opponentPokemonName := "(Opponent)"

	if c.PlayerSquad != nil && c.PlayerActiveIdx >= 0 && c.PlayerActiveIdx < len(c.PlayerSquad) {
		poke := c.PlayerSquad[c.PlayerActiveIdx]
		if poke != nil && poke.Base != nil && c.PlayerActiveIdx < len(c.PlayerMaxHPs) {
			yourPokemonName = poke.Base.Name
			maxHP := c.PlayerMaxHPs[c.PlayerActiveIdx]
			if maxHP > 0 {
				yourHpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
			}
		} else {
			log.Printf("Warning: Could not get player active Pokemon data (Index: %d, Squad Len: %d)", c.PlayerActiveIdx, len(c.PlayerSquad))
		}
	}

	if c.EnemySquad != nil && c.EnemyActiveIdx >= 0 && c.EnemyActiveIdx < len(c.EnemySquad) {
		poke := c.EnemySquad[c.EnemyActiveIdx]
		if poke != nil && poke.Base != nil && c.EnemyActiveIdx < len(c.EnemyMaxHPs) {
			opponentPokemonName = poke.Base.Name
			maxHP := c.EnemyMaxHPs[c.EnemyActiveIdx]
			if maxHP > 0 {
				oppHpPercent = math.Max(0, math.Min(100, (poke.CurrentHP/maxHP)*100.0))
			}
		} else {
			log.Printf("Warning: Could not get opponent active Pokemon data (Index: %d, Squad Len: %d)", c.EnemyActiveIdx, len(c.EnemySquad))
		}
	}

	fmt.Printf("\n---  Your \033[1m%s:\033[0m %.1f%% HP | Enemy \033[1m%s:\033[0m%.1f%% HP ---\n",
		strings.Title(yourPokemonName),
		yourHpPercent,
		strings.Title(opponentPokemonName),
		oppHpPercent,
	)
	fmt.Println("===================")
}

func (c *Client) handleOpponentDisconnected(msg Message) {
	opponentName, _ := msg.Message["opponent"].(string)
	if opponentName == "" {
		opponentName = "Opponent"
	}
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
	if c.AwaitingForcedSwitch {
		var targetIndex int = -1
		if switchNum, err := strconv.Atoi(command); err == nil && len(parts) == 1 {
			if switchNum < 1 || switchNum > 6 {
				fmt.Println("Switch number must be between 1 and 6.")
				fmt.Print("Enter the number of the Pokemon to switch to: ")
				return
			}
			targetIndex = switchNum - 1
		} else if (command == "switch" || command == "s") && len(parts) == 2 {
			actionIndex, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid number format for index.")
				fmt.Print("Enter the number of the Pokemon to switch to: ")
				return
			}
			if actionIndex < 1 || actionIndex > 6 {
				fmt.Println("Switch index must be between 1 and 6.")
				fmt.Print("Enter the number of the Pokemon to switch to: ")
				return
			}
			targetIndex = actionIndex - 1
		} else {
			fmt.Println("Invalid command. Use 'switch <number>' or just the number.")
			fmt.Print("Enter the number of the Pokemon to switch to: ")
			return
		}
		if targetIndex == c.PlayerActiveIdx {
			fmt.Println("Cannot switch to the Pokemon that is already active.")
			fmt.Print("Enter the number of the Pokemon to switch to: ")
			return
		}
		if c.PlayerSquad != nil && targetIndex >= 0 && targetIndex < len(c.PlayerSquad) {
			targetPoke := c.PlayerSquad[targetIndex]
			if targetPoke == nil {
				fmt.Println("Error: Invalid Pokemon data for switch target.")
				fmt.Print("Enter the number of the Pokemon to switch to: ")
				return
			}
			if targetPoke.Fainted {
				fmt.Printf("Cannot switch to %s because it has fainted.\n", targetPoke.Base.Name)
				fmt.Print("Enter the number of the Pokemon to switch to: ")
				return
			}
			c.sendSwitchAction(targetIndex)
		} else {
			fmt.Println("Invalid switch index.")
			fmt.Print("Enter the number of the Pokemon to switch to: ")
		}
	} else {
		isFainted := false
		if c.PlayerSquad != nil && c.PlayerActiveIdx >= 0 && c.PlayerActiveIdx < len(c.PlayerSquad) {
			activePoke := c.PlayerSquad[c.PlayerActiveIdx]
			if activePoke != nil {
				isFainted = activePoke.Fainted
			}
		}
		if !isFainted && len(parts) == 1 {
			if moveNum, err := strconv.Atoi(parts[0]); err == nil {
				moveIndex := moveNum - 1
				if c.LastAvailableMovesInfo == nil || moveIndex < 0 || moveIndex >= len(c.LastAvailableMovesInfo) {
					fmt.Println("Invalid move number or move info unavailable.")
					fmt.Print("Enter your action: ")
					return
				}
				selectedMove := c.LastAvailableMovesInfo[moveIndex]
				if selectedMove.CurrentPP <= 0 {
					fmt.Printf("Move '%s' has no PP left!\n", selectedMove.Name)
					fmt.Print("Enter your action: ")
					return
				}
				c.sendGameAction("move", moveNum, 0)
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
			if isFainted {
				fmt.Println("Your active Pokemon has fainted. You must switch.")
				fmt.Print("Enter action (switch <number>): ")
				return
			}
			if actionIndex < 1 || actionIndex > 4 {
				fmt.Println("Move index must be between 1 and 4.")
				fmt.Print("Enter your action: ")
				return
			}
			moveIndex0Based := actionIndex - 1
			if c.LastAvailableMovesInfo == nil || moveIndex0Based < 0 || moveIndex0Based >= len(c.LastAvailableMovesInfo) {
				fmt.Println("Invalid move number or move info unavailable.")
				fmt.Print("Enter your action: ")
				return
			}
			selectedMove := c.LastAvailableMovesInfo[moveIndex0Based]
			if selectedMove.CurrentPP <= 0 {
				fmt.Printf("Move '%s' has no PP left!\n", selectedMove.Name)
				fmt.Print("Enter your action: ")
				return
			}
			c.sendGameAction("move", actionIndex, 0)
		case "switch", "s":
			if actionIndex < 1 || actionIndex > 6 {
				fmt.Println("Switch index must be between 1 and 6.")
				fmt.Print("Enter your action: ")
				return
			}
			targetIndex := actionIndex - 1
			if targetIndex == c.PlayerActiveIdx {
				fmt.Println("Cannot switch to the Pokemon that is already active.")
				fmt.Print("Enter your action: ")
				return
			}
			if c.PlayerSquad != nil && targetIndex >= 0 && targetIndex < len(c.PlayerSquad) {
				targetPoke := c.PlayerSquad[targetIndex]
				if targetPoke == nil {
					fmt.Println("Error: Invalid target Pokemon data.")
					fmt.Print("Enter your action: ")
					return
				}
				if targetPoke.Fainted {
					fmt.Printf("Cannot switch to %s because it has fainted.\n", targetPoke.Base.Name)
					fmt.Print("Enter your action: ")
					return
				}
			} else {
				fmt.Println("Invalid switch index.")
				fmt.Print("Enter your action: ")
				return
			}
			c.sendGameAction("switch", 0, targetIndex)
		default:
			fmt.Println("Unknown command. Use 'move <number>' or 'switch <number>'.")
			fmt.Print("Enter your action: ")
		}
	}
}

func (c *Client) sendGameAction(actionType string, moveIndex, switchIndex int) {
	actionStr := fmt.Sprintf("%s|%s|%d|%d", GameActionMarker, actionType, moveIndex, switchIndex)
	log.Printf("Sending game action: %s", actionStr)
	if c.Conn == nil || !c.Connected {
		log.Println("Error: Cannot send game action, not connected.")
		fmt.Println("Error: Connection lost.")
		c.Disconnect()
		return
	}
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Conn.Write([]byte(actionStr))
	c.Conn.SetWriteDeadline(time.Time{})
	if err != nil {
		log.Printf("Failed to send game action: %v", err)
		fmt.Println("Error sending action. Disconnecting.")
		c.Disconnect()
	} else {
		fmt.Printf("Sent %s action to server...\n", actionType)
	}
}
