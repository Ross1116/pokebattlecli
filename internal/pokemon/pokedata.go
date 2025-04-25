package pokemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

var (
	cache     = make(map[string][]byte)
	cacheLock = sync.RWMutex{}
	cacheTTL  = make(map[string]time.Time)
)

func FetchData[T any](url string, result *T) error {
	cacheLock.RLock()
	cachedData, exists := cache[url]
	expiryTime, timeExists := cacheTTL[url]
	cacheLock.RUnlock()

	if exists && timeExists && time.Now().Before(expiryTime) {
		return json.Unmarshal(cachedData, result)
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	cacheLock.Lock()
	cache[url] = body
	cacheTTL[url] = time.Now().Add(1 * time.Hour)
	cacheLock.Unlock()

	return json.Unmarshal(body, result)
}

func FetchPokemonData(identifier any) (*Pokemon, error) {
	var url string
	switch v := identifier.(type) {
	case string:
		url = fmt.Sprintf("http://pokeapi.co/api/v2/pokemon/%s/", v)
	case int:
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

func FetchMovesInParallel(moveURLs []string) ([]*MoveInfo, error) {
	moves := make([]*MoveInfo, len(moveURLs))
	var wg sync.WaitGroup
	errChan := make(chan error, len(moveURLs))

	semaphore := make(chan struct{}, 10)
	var resultLock sync.Mutex

	for i, url := range moveURLs {
		wg.Add(1)
		go func(idx int, moveURL string) {
			defer wg.Done()

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

	select {
	case err := <-errChan:
		return nil, err
	default:
		return moves, nil
	}
}
