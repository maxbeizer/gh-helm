package pillars

type Summary struct {
	Counts    map[string]int `json:"counts"`
	Highlights []string       `json:"highlights"`
}

func BuildSummary(matches []PillarMatch) Summary {
	counts := map[string]int{}
	for _, match := range matches {
		counts[match.Pillar]++
	}
	return Summary{Counts: counts}
}
