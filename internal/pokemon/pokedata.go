package pokemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Shared HTTP client for connection pooling
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

// Cache implementation
var (
	cache     = make(map[string][]byte)
	cacheLock = sync.RWMutex{}
	cacheTTL  = make(map[string]time.Time)
)

// Original function signature maintained
func FetchData[T any](url string, result *T) error {
	// Check cache first
	cacheLock.RLock()
	cachedData, exists := cache[url]
	expiryTime, timeExists := cacheTTL[url]
	cacheLock.RUnlock()

	if exists && timeExists && time.Now().Before(expiryTime) {
		// Use cached data
		return json.Unmarshal(cachedData, result)
	}

	// Cache miss, fetch from network using shared client
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Store in cache
	cacheLock.Lock()
	cache[url] = body
	cacheTTL[url] = time.Now().Add(1 * time.Hour) // Cache for 1 hour
	cacheLock.Unlock()

	return json.Unmarshal(body, result)
}

// Original function signature maintained
func FetchPokemonData(identifier any) (*Pokemon, error) {
	var url string
	switch v := identifier.(type) {
	case string:
		// Fetch by name
		url = fmt.Sprintf("http://pokeapi.co/api/v2/pokemon/%s/", v)
	case int:
		// Fetch by ID
		url = fmt.Sprintf("http://pokeapi.co/api/v2/pokemon/%d/", v)
	default:
		return nil, fmt.Errorf("invalid identifier type")
	}

	var pokemon Pokemon
	err := FetchData(url, &pokemon)
	if err != nil {
		return nil, err
	}

	return &pokemon, nil
}

// Original function signature maintained
func FetchMoveData(url string) (*MoveInfo, error) {
	var move MoveInfo
	err := FetchData(url, &move)
	if err != nil {
		return nil, err
	}

	return &move, nil
}

func FetchMoveByName(name string) (*MoveInfo, error) {
	url := fmt.Sprintf("https://pokeapi.co/api/v2/move/%s/", name)
	return FetchMoveData(url)
}

// Helper function to fetch moves concurrently
// This doesn't change the original functions but can be used with them
func fetchMovesInParallel(moveURLs []string) ([]*MoveInfo, error) {
	moves := make([]*MoveInfo, len(moveURLs))
	var wg sync.WaitGroup
	errChan := make(chan error, len(moveURLs))

	// Limit concurrency
	semaphore := make(chan struct{}, 10)
	var resultLock sync.Mutex

	for i, url := range moveURLs {
		wg.Add(1)
		go func(idx int, moveURL string) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			move, err := FetchMoveData(moveURL)
			if err != nil {
				errChan <- err
				return
			}

			resultLock.Lock()
			moves[idx] = move
			resultLock.Unlock()
		}(i, url)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	select {
	case err := <-errChan:
		return nil, err
	default:
		return moves, nil
	}
}
