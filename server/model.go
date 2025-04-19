package server

import "net"

type Server struct {
	host    string
	port    string
	clients map[string]*Client
	lobbies map[string]*Lobby
}

type Client struct {
	conn     net.Conn
	username string
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
