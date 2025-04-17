package pokemon

type Pokemon struct {
	ID      int         `json:"id"`
	Name    string      `json:"name"`
	Types   []TypeSlot  `json:"types"`
	Stats   []BaseStats `json:"stats"`
	Moves   []MoveSlot  `json:"moves"`
	Fainted bool        `json:"fainted"`
}

type BaseStats struct {
	BaseStat int         `json:"base_stat"`
	Stat     ApiResource `json:"stat"`
}

type TypeSlot struct {
	Slot int      `json:"slot"`
	Type TypeInfo `json:"type"`
}

type TypeInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type MoveSlot struct {
	Move                ApiResource              `json:"move"`
	VersionGroupDetails []VersionGroupDetailInfo `json:"version_group_details"`
}

type ApiResource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type VersionGroupDetailInfo struct {
	LevelLearnedAt  int         `json:"level_learned_at"`
	MoveLearnMethod ApiResource `json:"move_learn_method"`
	VersionGroup    ApiResource `json:"version_group"`
}

type MoveInfo struct {
	Accuracy      int             `json:"accuracy"`
	DamageClass   ApiResource     `json:"damage_class"`
	EffectChance  int             `json:"effect_chance"`
	EffectEntries []EffectEntries `json:"effect_entries"`
	Name          string          `json:"name"`
	Power         int             `json:"power"`
	Pp            int             `json:"pp"`
	Priority      int             `json:"priority"`
	Type          ApiResource     `json:"type"`
}

type EffectEntries struct {
	Effect      string      `json:"effect"`
	Language    ApiResource `json:"language"`
	ShortEffect string      `json:"short_effect"`
}
