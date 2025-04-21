package server

import (
	"net"
	"sync"
)

type Server struct {
	host    string
	port    string
	clients map[string]*Client
	Lobbies map[string]*Lobby
	mu      sync.RWMutex
}

type Client struct {
	Conn     net.Conn
	Username string

	startGameSignal chan struct{}
	endGameSignal   chan struct{}

	gameActionChan chan []byte
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

type Request struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}

func NewClient(conn net.Conn, username string) *Client {
	return &Client{
		Conn:            conn,
		Username:        username,
		startGameSignal: make(chan struct{}),
		endGameSignal:   make(chan struct{}),
		gameActionChan:  make(chan []byte, 5),
	}
}

