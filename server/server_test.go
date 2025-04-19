package server_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	serverpkg "github.com/ross1116/pokebattlecli/server"
)

func TestServerRun(t *testing.T) {
	srv := serverpkg.New(&serverpkg.Config{
		Host: "localhost",
		Port: "3333",
	})
	go srv.Run()
	time.Sleep(200 * time.Millisecond)
}

func TestServerHandlesClientRegistration(t *testing.T) {
	config := &serverpkg.Config{Host: "localhost", Port: "9090"}
	srv := serverpkg.New(config)
	go srv.Run()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", config.Host+":"+config.Port)
	if err != nil {
		t.Fatalf("Client failed to connect: %v", err)
	}
	defer conn.Close()

	msg := map[string]string{
		"type":     "register",
		"username": "testuser",
	}
	payload, _ := json.Marshal(msg)
	_, err = conn.Write(payload)
	if err != nil {
		t.Fatalf("Client failed to write: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	client, ok := srv.Clients()["testuser"]
	if !ok {
		t.Fatalf("Expected client 'testuser' to be registered")
	}
	if client.Username() != "testuser" {
		t.Fatalf("Expected username to be 'testuser', got '%s'", client.Username())
	}
}

func TestServerHandlesGetPlayers(t *testing.T) {
	config := &serverpkg.Config{Host: "localhost", Port: "9091"}
	srv := serverpkg.New(config)
	go srv.Run()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", config.Host+":"+config.Port)
	if err != nil {
		t.Fatalf("Client failed to connect: %v", err)
	}
	defer conn.Close()

	registerMsg := map[string]string{
		"type":     "register",
		"username": "player1",
	}
	regPayload, _ := json.Marshal(registerMsg)
	conn.Write(regPayload)
	time.Sleep(100 * time.Millisecond)

	getPlayersMsg := map[string]string{
		"type": "get_players",
	}
	payload, _ := json.Marshal(getPlayersMsg)
	conn.Write(payload)
}

func TestServerHandlesMatchmake(t *testing.T) {
	config := &serverpkg.Config{Host: "localhost", Port: "9092"}
	srv := serverpkg.New(config)
	go srv.Run()
	time.Sleep(100 * time.Millisecond)

	conn1, err := net.Dial("tcp", config.Host+":"+config.Port)
	if err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", config.Host+":"+config.Port)
	if err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer conn2.Close()

	msg1 := map[string]string{
		"type":     "register",
		"username": "player1",
	}
	payload1, _ := json.Marshal(msg1)
	conn1.Write(payload1)
	time.Sleep(100 * time.Millisecond)

	msg2 := map[string]string{
		"type":     "register",
		"username": "player2",
	}
	payload2, _ := json.Marshal(msg2)
	conn2.Write(payload2)
	time.Sleep(100 * time.Millisecond)

	matchReq := map[string]string{
		"type":     "matchmake",
		"username": "player1",
		"opponent": "player2",
	}
	matchPayload, _ := json.Marshal(matchReq)
	conn1.Write(matchPayload)
	time.Sleep(100 * time.Millisecond)

	if _, ok := srv.Clients()["player1"]; !ok {
		t.Fatalf("Expected player1 to be registered")
	}
	if _, ok := srv.Clients()["player2"]; !ok {
		t.Fatalf("Expected player2 to be registered")
	}
}
