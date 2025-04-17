package pokemon

import "math/rand"

func SelectRandSquad() []*Pokemon {
	var squad []*Pokemon
	for range 6 {
		randNum := rand.Intn(386) + 1
		poke, _ := FetchPokemonData(randNum)
		squad = append(squad, poke)
	}

	return squad
}

