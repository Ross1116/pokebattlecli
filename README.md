# PokéBattleCLI

A command-line interface (CLI) application for simulating Pokémon battles between two players using a client-server architecture.

## Description

This project allows users to connect to a central server, find other players, and engage in text-based Pokémon battles. The server manages game state and battle logic, while the client provides the user interface for interacting with the server and the battle. Pokémon and move data are fetched dynamically (likely from an external source like PokeAPI).

## Features

* **Client-Server Architecture:** Uses TCP sockets for communication between clients and the server.
* **Player Management:** User registration and listing of currently connected players.
* **Matchmaking:** Allows players to challenge each other to battles.
* **Turn-Based Battles:** Simulates Pokémon battles turn by turn.
* **Dynamic Data:** Fetches Pokémon base stats and move details.
* **Battle Mechanics:** Implements core mechanics like:
    * HP calculation and tracking.
    * Move execution and damage calculation.
    * PP (Power Points) tracking and validation.
    * Switching Pokémon.
    * Fainting condition.
    * Status effects (basic implementation).
* **Text-Based Interface:** All interaction happens through the command line.

## Architecture

* **Server (`cmd/server/`):** Handles client connections, manages player lists and lobbies, orchestrates battles, and enforces game rules.
* **Client (`cmd/client/`):** Connects to the server, sends user commands (registration, matchmaking, battle actions), receives updates from the server, and displays game information and battle progress.
* **Internal Packages (`internal/`):** Contain shared logic for battle mechanics (`battle`), Pokémon/move data fetching and structures (`pokemon`, `stats`), etc.

## Setup and Running

**Prerequisites:**
* Go programming language environment (e.g., Go 1.18 or later).

**Build:**
(Optional) You can build the executables:
```bash
go build -o server_app ./cmd/server/
go build -o client_app ./cmd/client/
Running the Server:Navigate to the project root directory.Run the server using go run or the compiled binary:go run ./cmd/server/
# or ./server_app
The server will log that it has started, usually on localhost:1234 (or configured host/port).Running the Client:Open a new terminal window.Navigate to the project root directory.Run the client using go run or the compiled binary, providing a unique username:go run ./cmd/client/ -user <your_username>
# Example: go run ./cmd/client/ -user Ash
# or ./client_app -user Ash
Repeat step 3 in another terminal for a second player with a different username.
```
## Gameplay (Client Commands)

### Once connected, use the following commands in the client terminal:

- `help`: Displays available commands.
  
- `players`: Lists currently online players.
 
- `match <username>`: Challenges the specified player to a battle.
  
- `quit`: Disconnects from the server and exits the client.

### During a Battle:
- Follow the prompts to enter actions.
 
- `move <number>` or `<number>`: Use the move corresponding to the number shown (e.g., move 1 or just 1). Performs moves based on available PP.
  
- `switch <number>`: Switch to the Pokémon corresponding to the number in your squad list. Cannot switch to fainted Pokémon or the currently
