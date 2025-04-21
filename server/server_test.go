package server_test

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/ross1116/pokebattlecli/server"
)

type MockConn struct {
	buffer *bytes.Buffer
}

func (m *MockConn) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *MockConn) Read(p []byte) (n int, err error) {
	return m.buffer.Read(p)
}

func (m *MockConn) Close() error                       { return nil }
func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

func newMockConn() *MockConn {
	return &MockConn{buffer: &bytes.Buffer{}}
}

func TestHandleMatchmake(t *testing.T) {
	serverInstance := server.New(&server.Config{
		Host: "localhost",
		Port: "1234",
	})

	client1Conn := newMockConn()
	client2Conn := newMockConn()

	client1 := &server.Client{
		Username: "player1",
		Conn:     client1Conn,
	}
	client2 := &server.Client{
		Username: "player2",
		Conn:     client2Conn,
	}

	serverInstance.AddClient("player1", client1)
	serverInstance.AddClient("player2", client2)

	matchmakeMessage := map[string]string{
		"type":     "matchmake",
		"username": "player1",
		"opponent": "player2",
	}

	// Simulate player1 sending the matchmake request
	serverInstance.HandleMatchmake(matchmakeMessage, client1Conn)

	// Validate player1 response
	var response1 map[string]interface{}
	err := json.Unmarshal(client1Conn.buffer.Bytes(), &response1)
	if err != nil {
		t.Fatalf("Error decoding response for player1: %v", err)
	}

	if response1["type"] != "match_start" {
		t.Errorf("Expected 'match_start', got %v", response1["type"])
	}

	if opponent, ok := response1["message"].(map[string]interface{})["opponent"]; !ok || opponent != "player2" {
		t.Errorf("Expected opponent 'player2', got %v", opponent)
	}

	// Validate player2 response
	var response2 map[string]interface{}
	err = json.Unmarshal(client2Conn.buffer.Bytes(), &response2)
	if err != nil {
		t.Fatalf("Error decoding response for player2: %v", err)
	}

	if response2["type"] != "match_start" {
		t.Errorf("Expected 'match_start', got %v", response2["type"])
	}

	if opponent, ok := response2["message"].(map[string]interface{})["opponent"]; !ok || opponent != "player1" {
		t.Errorf("Expected opponent 'player1', got %v", opponent)
	}
}

