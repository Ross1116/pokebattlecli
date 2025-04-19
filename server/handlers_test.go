package server_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	serverpkg "github.com/ross1116/pokebattlecli/server"
)

func setupTestServer(t *testing.T, port string) *serverpkg.Server {
	t.Helper()
	config := &serverpkg.Config{
		Host: "localhost",
		Port: port,
	}
	srv := serverpkg.New(config)
	go srv.Run()
	time.Sleep(200 * time.Millisecond) // give the server time to start
	return srv
}

func sendMessage(t *testing.T, conn net.Conn, msg map[string]string) {
	t.Helper()
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}
	_, err = conn.Write(payload)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
}

func TestHandleRegistration(t *testing.T) {
	srv := setupTestServer(t, "9001")
	conn, err := net.Dial("tcp", "localhost:9001")
	if err != nil {
		t.Fatalf("Client failed to connect: %v", err)
	}
	defer conn.Close()

	msg := map[string]string{
		"type":     "register",
		"username": "testuser1",
	}
	sendMessage(t, conn, msg)

	time.Sleep(100 * time.Millisecond)

	if _, ok := srv.Clients()["testuser1"]; !ok {
		t.Fatalf("Expected 'testuser1' to be registered")
	}
}

func TestHandleGetPlayers(t *testing.T) {
	srv := setupTestServer(t, "9002")
	conn1, _ := net.Dial("tcp", "localhost:9002")
	defer conn1.Close()
	sendMessage(t, conn1, map[string]string{
		"type":     "register",
		"username": "user1",
	})

	conn2, _ := net.Dial("tcp", "localhost:9002")
	defer conn2.Close()
	sendMessage(t, conn2, map[string]string{
		"type":     "register",
		"username": "user2",
	})

	time.Sleep(100 * time.Millisecond)

	sendMessage(t, conn2, map[string]string{
		"type": "get_players",
	})

	// we could read response too, but for now we just verify the server's state
	clients := srv.Clients()
	if len(clients) != 2 {
		t.Fatalf("Expected 2 registered clients, got %d", len(clients))
	}
}

func TestHandleMatchmake(t *testing.T) {
	srv := setupTestServer(t, "9003")

	conn1, _ := net.Dial("tcp", "localhost:9003")
	defer conn1.Close()
	sendMessage(t, conn1, map[string]string{
		"type":     "register",
		"username": "playerA",
	})

	conn2, _ := net.Dial("tcp", "localhost:9003")
	defer conn2.Close()
	sendMessage(t, conn2, map[string]string{
		"type":     "register",
		"username": "playerB",
	})

	time.Sleep(100 * time.Millisecond)

	sendMessage(t, conn1, map[string]string{
		"type":     "matchmake",
		"username": "playerA",
		"opponent": "playerB",
	})

	time.Sleep(100 * time.Millisecond)

	lobbyA, okA := srv.Lobbies()["playerA"]
	lobbyB, okB := srv.Lobbies()["playerB"]
	if !okA || !okB || lobbyA != lobbyB {
		t.Fatalf("Expected both players to be matched into the same lobby")
	}
}
