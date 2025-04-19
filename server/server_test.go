package server_test

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ross1116/pokebattlecli/server"
)

// mockConn implements the net.Conn interface for testing
type mockConn struct {
	readData  chan []byte
	writeData chan []byte
	closed    bool
}

func newMockConn() *mockConn {
	return &mockConn{
		readData:  make(chan []byte, 10),
		writeData: make(chan []byte, 10),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	data := <-m.readData
	copy(b, data)
	return len(data), nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, net.ErrClosed
	}
	m.writeData <- b
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

// Required methods for net.Conn interface
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestNewServer(t *testing.T) {
	config := &server.Config{
		Host: "localhost",
		Port: "8080",
	}
	s := server.New(config)

	if s.Clients() == nil {
		t.Error("Expected clients map to be initialized")
	}
	if s.Lobbies == nil {
		t.Error("Expected Lobbies map to be initialized")
	}
}

func TestHandleRegistration(t *testing.T) {
	s := server.New(&server.Config{Host: "localhost", Port: "8080"})
	conn := newMockConn()

	// Test registration with valid username
	msg := map[string]string{
		"type":     "register",
		"username": "player1",
	}

	go s.HandleRegistration(msg, conn)

	// Read the response
	select {
	case response := <-conn.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Type != "registration" {
			t.Errorf("Expected response type 'registration', got %s", resp.Type)
		}

		status, ok := resp.Message["status"].(string)
		if !ok || !strings.Contains(status, "registered successfully") {
			t.Errorf("Expected success message, got %v", resp.Message["status"])
		}

		if len(s.Clients()) != 1 {
			t.Errorf("Expected 1 client, got %d", len(s.Clients()))
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for response")
	}

	// Test reconnection
	conn2 := newMockConn()
	go s.HandleRegistration(msg, conn2)

	select {
	case response := <-conn2.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Type != "reconnect" {
			t.Errorf("Expected response type 'reconnect', got %s", resp.Type)
		}

		if len(s.Clients()) != 1 {
			t.Errorf("Expected still 1 client, got %d", len(s.Clients()))
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for reconnection response")
	}
}

func TestHandleGetPlayers(t *testing.T) {
	s := server.New(&server.Config{Host: "localhost", Port: "8080"})
	conn1 := newMockConn()
	conn2 := newMockConn()

	// Register two players
	msg1 := map[string]string{"type": "register", "username": "player1"}
	msg2 := map[string]string{"type": "register", "username": "player2"}

	s.HandleRegistration(msg1, conn1)
	s.HandleRegistration(msg2, conn2)

	// Clear any registration responses
	<-conn1.writeData
	<-conn2.writeData

	// Test get players
	conn := newMockConn()
	go s.HandleGetPlayers(conn)

	select {
	case response := <-conn.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Type != "player_list" {
			t.Errorf("Expected response type 'player_list', got %s", resp.Type)
		}

		playersInterface, ok := resp.Message["players"]
		if !ok {
			t.Fatal("Expected 'players' field in response")
		}

		players, ok := playersInterface.([]interface{})
		if !ok {
			t.Fatalf("Expected players to be an array, got %T", playersInterface)
		}

		if len(players) != 2 {
			t.Errorf("Expected 2 players, got %d", len(players))
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for player list response")
	}
}

func TestHandleMatchmake(t *testing.T) {
	s := server.New(&server.Config{Host: "localhost", Port: "8080"})
	conn1 := newMockConn()
	conn2 := newMockConn()

	// Register two players
	msg1 := map[string]string{"type": "register", "username": "player1"}
	msg2 := map[string]string{"type": "register", "username": "player2"}

	s.HandleRegistration(msg1, conn1)
	s.HandleRegistration(msg2, conn2)

	// Clear any registration responses
	<-conn1.writeData
	<-conn2.writeData

	// Test matchmaking
	msg := map[string]string{
		"type":     "matchmake",
		"username": "player1",
		"opponent": "player2",
	}

	go s.HandleMatchmake(msg, conn1)

	// Check player1 response
	select {
	case response := <-conn1.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal player1 response: %v", err)
		}

		if resp.Type != "match_start" {
			t.Errorf("Expected response type 'match_start', got %s", resp.Type)
		}

		opponent, ok := resp.Message["opponent"].(string)
		if !ok || opponent != "player2" {
			t.Errorf("Expected opponent to be 'player2', got %v", resp.Message["opponent"])
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for player1 match response")
	}

	// Check player2 response
	select {
	case response := <-conn2.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal player2 response: %v", err)
		}

		if resp.Type != "match_start" {
			t.Errorf("Expected response type 'match_start', got %s", resp.Type)
		}

		opponent, ok := resp.Message["opponent"].(string)
		if !ok || opponent != "player1" {
			t.Errorf("Expected opponent to be 'player1', got %v", resp.Message["opponent"])
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for player2 match response")
	}

	// Check game end messages
	// Player 1 (win)
	select {
	case response := <-conn1.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal player1 game end response: %v", err)
		}

		if resp.Type != "game_end" {
			t.Errorf("Expected response type 'game_end', got %s", resp.Type)
		}

		result, ok := resp.Message["result"].(string)
		if !ok || result != "win" {
			t.Errorf("Expected result to be 'win', got %v", resp.Message["result"])
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for player1 game end response")
	}

	// Player 2 (lose)
	select {
	case response := <-conn2.writeData:
		var resp server.Response
		if err := json.Unmarshal(response, &resp); err != nil {
			t.Fatalf("Failed to unmarshal player2 game end response: %v", err)
		}

		if resp.Type != "game_end" {
			t.Errorf("Expected response type 'game_end', got %s", resp.Type)
		}

		result, ok := resp.Message["result"].(string)
		if !ok || result != "lose" {
			t.Errorf("Expected result to be 'lose', got %v", resp.Message["result"])
		}

	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for player2 game end response")
	}

	// Verify lobbies are cleaned up
	if len(s.Lobbies) != 0 {
		t.Errorf("Expected lobbies to be empty after game end, got %d lobbies", len(s.Lobbies))
	}
}

func TestHandleDisconnection(t *testing.T) {
	s := server.New(&server.Config{Host: "localhost", Port: "8080"})
	conn1 := newMockConn()
	conn2 := newMockConn()

	// Register players
	msg1 := map[string]string{"type": "register", "username": "player1"}
	msg2 := map[string]string{"type": "register", "username": "player2"}

	s.HandleRegistration(msg1, conn1)
	s.HandleRegistration(msg2, conn2)

	// Clear registration responses
	<-conn1.writeData
	<-conn2.writeData

	// Create a matchmaking request to form a lobby
	matchMsg := map[string]string{
		"type":     "matchmake",
		"username": "player1",
		"opponent": "player2",
	}
	s.HandleMatchmake(matchMsg, conn1)

	// Clear match responses
	<-conn1.writeData
	<-conn2.writeData
	<-conn1.writeData
	<-conn2.writeData

	// Verify we have clients and lobbies before disconnection
	initialClientCount := len(s.Clients())
	if initialClientCount != 2 {
		t.Errorf("Expected 2 clients before disconnection, got %d", initialClientCount)
	}

	// Test disconnection
	s.HandleDisconnection(conn1)

	// Verify the client was removed
	if len(s.Clients()) != initialClientCount-1 {
		t.Errorf("Expected client count to decrease by 1 after disconnection")
	}

	// Verify the lobby was removed
	if len(s.Lobbies) != 0 {
		t.Error("Expected lobbies to be empty after disconnection")
	}
}

func TestPlayerAlreadyInMatch(t *testing.T) {
	s := server.New(&server.Config{Host: "localhost", Port: "8080"})
	conn1 := newMockConn()
	conn2 := newMockConn()
	conn3 := newMockConn()

	// Register three players
	s.HandleRegistration(map[string]string{"type": "register", "username": "player1"}, conn1)
	s.HandleRegistration(map[string]string{"type": "register", "username": "player2"}, conn2)
	s.HandleRegistration(map[string]string{"type": "register", "username": "player3"}, conn3)

	// Clear registration responses
	<-conn1.writeData
	<-conn2.writeData
	<-conn3.writeData

	// First, let's modify the server code to make this test possible
	// We need to modify IsInLobby in handlers.go to check by username:
	//
	// func (server *Server) IsInLobby(client *Client) bool {
	//     _, exists := server.Lobbies[client.Username]
	//     return exists
	// }

	// Manually set up a lobby by adding entries to the Lobbies map
	fakelobby := &server.Lobby{}
	s.Lobbies["player1"] = fakelobby
	s.Lobbies["player2"] = fakelobby

	// Try to match player3 with player2 (who is manually placed in a lobby)
	matchMsg := map[string]string{
		"type":     "matchmake",
		"username": "player3",
		"opponent": "player2",
	}
	s.HandleMatchmake(matchMsg, conn3)

	// No response should be sent (matchmaking should fail)
	select {
	case response := <-conn3.writeData:
		t.Errorf("Expected no response for player3, but got: %s", string(response))
	case <-time.After(100 * time.Millisecond):
		// This is the expected behavior
	}
}
