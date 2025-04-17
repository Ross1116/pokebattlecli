package stats

import "github.com/ross1116/pokebattlecli/internal/pokemon"

func HpCalc(baseHp int) float64 {
	iv := 31
	level := 100
	return float64(((2*baseHp+iv+31)*level)/100 + 10)
}

func StatCalc(baseStat int) float64 {
	iv := 31
	level := 100
	return float64(((2*baseStat+iv+31)*level)/100 + 5)
}

func GetStat(p *pokemon.Pokemon, statName string) int {
	for _, stat := range p.Stats {
		if stat.Stat.Name == statName {
			return stat.BaseStat
		}
	}
	return 0
}
