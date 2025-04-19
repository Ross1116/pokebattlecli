package client

import (
	"encoding/json"
	"net"
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
}

type Message struct {
	Type    string                 `json:"type"`
	Message map[string]interface{} `json:"message"`
}

func Marshal(data map[string]string) ([]byte, error) {
	return json.Marshal(data)
}

