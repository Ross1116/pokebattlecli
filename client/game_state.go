package client

type PokemonStateInfo struct {
	SquadIndex int     `json:"squad_index"`
	Name       string  `json:"name"`
	CurrentHP  float64 `json:"current_hp"`
	MaxHP      float64 `json:"max_hp"`
	HPPercent  float64 `json:"hp_percent"`
	Fainted    bool    `json:"fainted"`
	Status     string  `json:"status"`
}
