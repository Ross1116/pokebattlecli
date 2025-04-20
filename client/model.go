package client

import (
	"encoding/json"
	"net"

	"github.com/ross1116/pokebattlecli/internal/battle"
	"github.com/ross1116/pokebattlecli/internal/pokemon"
)

type Config struct {
	ServerHost string
	ServerPort string
	Username   string
}

type Client struct {
	Config      *Config
	Conn        net.Conn
	Connected   bool
	Opponent    string
	InMatch     bool
	MessageChan chan Message

	// Game state
	GameActive       bool
	GameInputChannel chan string
	PlayerSquad      []*battle.BattlePokemon
	EnemySquad       []*battle.BattlePokemon
	PlayerMovesets   [][]pokemon.MoveInfo
	EnemyMovesets    [][]pokemon.MoveInfo
	PlayerActiveIdx  int
	EnemyActiveIdx   int
	PlayerMaxHPs     []float64
	EnemyMaxHPs      []float64
}

type Message struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}

func Marshal(data map[string]string) ([]byte, error) {
	return json.Marshal(data)
}
