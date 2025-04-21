package server

import (
	"net"

	"github.com/ross1116/pokebattlecli/internal/battle"
)

type Server struct {
	host    string
	port    string
	clients map[string]*Client
	Lobbies map[string]*Lobby
}

type Client struct {
	Conn     net.Conn
	Username string
}

type Lobby struct {
	player1 *Client
	player2 *Client
}

type Config struct {
	Host string
	Port string
}

type Response struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}

type PlayerView struct {
	YourTeam          []battle.PokemonSummary
	OpponentTeam      []battle.PokemonSummary
	AvailableMoves    []int
	CanSwitch         bool
	AvailableSwitches []int
	Weather           string
	FieldEffects      map[string]int
	TurnNumber        int
	LastActionResult  string
}

type Request struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}
