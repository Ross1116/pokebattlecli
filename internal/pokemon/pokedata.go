package pokemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func FetchData[T any](url string, result *T) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return err
	}

	return nil
}

func FetchPokemonData(identifier interface{}) (*Pokemon, error) {

	var url string
	switch v := identifier.(type) {
	case string:
		// Fetch by name
		url = fmt.Sprintf("http://localhost:4000/api/v2/pokemon/%s/", v)
	case int:
		// Fetch by ID
		url = fmt.Sprintf("http://localhost:4000/api/v2/pokemon/%d/", v)
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
