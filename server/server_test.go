package server_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	serverpkg "github.com/ross1116/pokebattlecli/server"
)

func TestServerRun(t *testing.T) {
	t.Log("Starting server...")
	srv := serverpkg.New(&serverpkg.Config{
		Host: "localhost",
		Port: "3333",
	})

	go srv.Run()

	t.Log("Waiting for server to start...")
	time.Sleep(200 * time.Millisecond)
	t.Log("Server should be running now.")
}

func TestServerHandlesClientRegistration(t *testing.T) {
	t.Log("Setting up server for testing...")
	config := &serverpkg.Config{Host: "localhost", Port: "9090"}
	srv := serverpkg.New(config)

	go srv.Run()
	t.Log("Server started. Waiting for client connection...")
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", config.Host+":"+config.Port)
	if err != nil {
		t.Fatalf("Client failed to connect: %v", err)
	}
	defer conn.Close()

	t.Log("Client connected. Sending registration message...")

	msg := map[string]string{
		"type":     "register",
		"username": "testuser",
	}
	payload, _ := json.Marshal(msg)
	_, err = conn.Write(payload)
	if err != nil {
		t.Fatalf("Client failed to write: %v", err)
	}

	t.Log("Sent registration request. Waiting for server to register client...")
	time.Sleep(100 * time.Millisecond)

	client, ok := srv.Clients()["testuser"]
	if !ok {
		t.Fatalf("Expected client 'testuser' to be registered")
	}

	t.Logf("Client 'testuser' found. Verifying username...")

	if client.Username() != "testuser" {
		t.Fatalf("Expected username to be 'testuser', got '%s'", client.Username())
	}

	t.Log("Test passed. Client successfully registered.")
}
