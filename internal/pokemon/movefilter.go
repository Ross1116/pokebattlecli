package pokemon

import (
	"log"
	"math/rand"
)

func FilterMoveByLearn(pokemon *Pokemon) []ApiResource {
	var moves []ApiResource

	for _, move := range pokemon.Moves {
		for _, detail := range move.VersionGroupDetails {
			if detail.MoveLearnMethod.Name == "level-up" || detail.MoveLearnMethod.Name == "egg" && detail.VersionGroup.Name == "firered-leafgreen" {
				moves = append(moves, move.Move)
			}
		}
	}

	return moves
}

func PickRandMoves(pokemon *Pokemon) []ApiResource {
	allMoves := FilterMoveByLearn(pokemon)

	moveSet := make(map[string]ApiResource)
	var uniqueMoves []ApiResource
	for _, move := range allMoves {
		if _, exists := moveSet[move.Name]; !exists {
			moveSet[move.Name] = move
			uniqueMoves = append(uniqueMoves, move)
		}
	}

	rand.Shuffle(len(uniqueMoves), func(i, j int) {
		uniqueMoves[i], uniqueMoves[j] = uniqueMoves[j], uniqueMoves[i]
	})

	var finalMoves []ApiResource
	for i := 0; i < len(uniqueMoves) && len(finalMoves) < 4; i++ {
		moveData, err := FetchMoveData(uniqueMoves[i].URL)
		if err != nil {
			log.Printf("failed to fetch move data for %s: %v", uniqueMoves[i].Name, err)
			continue
		}

		if moveData.DamageClass.Name != "status" {
			finalMoves = append(finalMoves, uniqueMoves[i])
		}
	}

	return finalMoves
}
func FilterStatusMoves(moves []ApiResource) []ApiResource {
	var filtered []ApiResource

	for _, move := range moves {
		moveData, err := FetchMoveData(move.URL)
		if err != nil {
			log.Printf("failed to fetch move data for %s: %v", move.Name, err)
			continue
		}

		if moveData.DamageClass.Name != "status" {
			filtered = append(filtered, move)
		}
	}

	return filtered
}

