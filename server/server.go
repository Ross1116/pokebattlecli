package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func New(config *Config) *Server {
	return &Server{
		host:    config.Host,
		port:    config.Port,
		clients: make(map[string]*Client),
		Lobbies: make(map[string]*Lobby),
	}
}

func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	log.Println("Server started on", server.host, ":", server.port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go server.HandleClient(conn)
	}
}

func (server *Server) SendResponse(conn net.Conn, response Response) {
	if conn == nil {
		log.Printf("Attempted to send response type %s to nil connection", response.Type)
		return
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response type %s: %v", response.Type, err)
		return
	}
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write(responseBytes)
	conn.SetWriteDeadline(time.Time{})
	if err != nil {
		log.Printf("Failed to send response type %s to %s: %v", response.Type, conn.RemoteAddr(), err)
	}
}

type clientMessage struct {
	data []byte
	err  error
}

func (server *Server) HandleClient(conn net.Conn) {
	var clientUsername string = ""
	var client *Client
	currentState := "PreGame"

	msgChan := make(chan clientMessage)
	stopReader := make(chan struct{})

	defer func() {
		log.Printf("Closing connection and stopping reader for %s (%s)", conn.RemoteAddr(), clientUsername)
		select {
		case <-stopReader:
		default:
			close(stopReader)
		}
		server.HandleDisconnection(conn, clientUsername)
		conn.Close()
	}()

	log.Printf("New connection established: %s. Starting reader goroutine.", conn.RemoteAddr())

	go func() {
		defer log.Printf("Reader goroutine stopped for %s (%s)", conn.RemoteAddr(), clientUsername)
		for {
			buf := make([]byte, 2048)
			n, err := conn.Read(buf)
			msg := clientMessage{}
			if n > 0 {
				dataCopy := make([]byte, n)
				copy(dataCopy, buf[:n])
				msg.data = dataCopy
			}
			msg.err = err
			select {
			case msgChan <- msg:
				if err != nil {
					return
				}
			case <-stopReader:
				return
			}
		}
	}()

	for {
		var gameStarted <-chan struct{} = nil
		var gameEnded <-chan struct{} = nil
		if client != nil {
			if currentState == "PreGame" {
				gameStarted = client.startGameSignal
			} else {
				gameEnded = client.endGameSignal
			}
		}

		select {
		case msg := <-msgChan:
			if msg.err != nil {
				return
			}

			if currentState == "PreGame" {
				var jsonMsg map[string]string
				if err := json.Unmarshal(msg.data, &jsonMsg); err != nil {
					log.Printf("Invalid JSON received from %s (%s) in PreGame state: %v. Message: %s", conn.RemoteAddr(), clientUsername, err, string(msg.data))
					continue
				}
				msgType, typeExists := jsonMsg["type"]
				if !typeExists {
					continue
				}
				switch msgType {
				case "register":
					usernameFromMsg, ok := jsonMsg["username"]
					if !ok || usernameFromMsg == "" {
						continue
					}
					server.mu.Lock()
					existingClient, exists := server.clients[usernameFromMsg]
					if exists {
						client = existingClient
					} else {
						client = NewClient(conn, usernameFromMsg)
						server.clients[usernameFromMsg] = client
					}
					clientUsername = usernameFromMsg
					server.mu.Unlock()
					server.HandleRegistration(jsonMsg, conn)
				case "get_players":
					if clientUsername == "" {
					} else {
						server.HandleGetPlayers(jsonMsg, conn)
					}
				case "matchmake":
					if clientUsername == "" {
					} else {
						jsonMsg["username"] = clientUsername
						server.HandleMatchmake(jsonMsg, conn)
					}
				default:
				}
			} else {
				if client == nil || client.gameActionChan == nil {
					log.Printf("Error: Received game data for %s but client/action channel is nil.", clientUsername)
					continue
				}
				log.Printf("HandleClient (%s): Forwarding data to gameActionChan: %s", clientUsername, string(msg.data))
				sendTimeout := time.After(2 * time.Second)
				select {
				case client.gameActionChan <- msg.data:
				case <-sendTimeout:
					log.Printf("Warning: Timeout forwarding game action from %s to runGameLoop.", clientUsername)
				}
			}

		case <-gameStarted:
			if currentState == "PreGame" {
				log.Printf("HandleClient for %s received game start signal. Transitioning state.", clientUsername)
				currentState = "InGame"
			} else {
			}

		case <-gameEnded:
			if currentState == "InGame" {
				log.Printf("HandleClient for %s received game end signal. Transitioning state.", clientUsername)
				currentState = "PreGame"
				if client != nil {
					client.startGameSignal = make(chan struct{})
					client.endGameSignal = make(chan struct{})
					client.gameActionChan = make(chan []byte, 5)
				}
			} else {
			}

		}
	}
}

