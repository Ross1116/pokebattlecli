package client

import (
	"encoding/json"
	"net"

	"github.com/ross1116/pokebattlecli/internal/battle"
)

type Config struct {
	ServerHost string
	ServerPort string
	Username   string
}

type MoveStateInfo struct {
	Name      string `json:"name"`
	CurrentPP int    `json:"current_pp"`
	MaxPP     int    `json:"max_pp"`
}

type Client struct {
	Config      *Config
	Conn        net.Conn
	Connected   bool
	Opponent    string
	InMatch     bool
	MessageChan chan Message

	GameActive             bool
	AwaitingForcedSwitch   bool
	PlayerSquad            []*battle.BattlePokemon
	EnemySquad             []*battle.BattlePokemon
	PlayerActiveIdx        int
	EnemyActiveIdx         int
	PlayerMaxHPs           []float64
	EnemyMaxHPs            []float64
	LastTurnDescription    []string
	LastAvailableMovesInfo []MoveStateInfo
}

type PlayerAction struct {
	Type          string `json:"type"`
	ActionIndex   int    `json:"actionIndex"`
	SwitchToIndex int    `json:"switchToIndex"`
}

type Message struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}

func Marshal(data map[string]string) ([]byte, error) {
	return json.Marshal(data)
}

const GameActionMarker = "GAME_ACTION_MARKER"
const SwitchActionMarker = "SWITCH_ACTION_MARKER"
