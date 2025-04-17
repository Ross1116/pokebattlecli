package stats

import "github.com/ross1116/pokebattlecli/internal/pokemon"

func HpCalc(baseHp int) float64 {
	return float64(baseHp*2 + 110)
}

func StatCalc(baseStat int) float64 {
	return float64(baseStat*2 + 5)
}

func GetStat(p *pokemon.Pokemon, statName string) int {
	for _, stat := range p.Stats {
		if stat.Stat.Name == statName {
			return stat.BaseStat
		}
	}
	return 0
}
