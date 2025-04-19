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

	batchSize := min(20, len(uniqueMoves))
	moveURLs := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		moveURLs[i] = uniqueMoves[i].URL
	}

	moveDetails, err := FetchMovesInParallel(moveURLs)
	if err != nil {
		log.Printf("Error in parallel fetch: %v", err)
		return fallbackSequentialFetch(uniqueMoves)
	}

	var finalMoves []ApiResource
	for i, moveDetail := range moveDetails {
		if moveDetail != nil && moveDetail.DamageClass.Name != "status" {
			finalMoves = append(finalMoves, uniqueMoves[i])
			if len(finalMoves) == 4 {
				break
			}
		}
	}

	if len(finalMoves) < 4 && batchSize < len(uniqueMoves) {
		remaining := fetchRemainingMoves(uniqueMoves[batchSize:], 4-len(finalMoves))
		finalMoves = append(finalMoves, remaining...)
	}

	return finalMoves
}

func fallbackSequentialFetch(uniqueMoves []ApiResource) []ApiResource {
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

func fetchRemainingMoves(remainingMoves []ApiResource, count int) []ApiResource {
	var additional []ApiResource
	for i := 0; i < len(remainingMoves) && len(additional) < count; i++ {
		moveData, err := FetchMoveData(remainingMoves[i].URL)
		if err != nil {
			log.Printf("failed to fetch move data for %s: %v", remainingMoves[i].Name, err)
			continue
		}
		if moveData.DamageClass.Name != "status" {
			additional = append(additional, remainingMoves[i])
		}
	}
	return additional
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

