package pillars

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/maxbeizer/gh-helm/internal/config"
)

type ActivityItem struct {
	Title     string
	Body      string
	Labels    []string
	Repo      string
	Files     []string
	URL       string
	Number    int
	CreatedAt time.Time
	ClosedAt  time.Time
}

type PillarMatch struct {
	Pillar     string `json:"pillar"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
	Item       ActivityItem `json:"-"`
}

func MapToPillars(item ActivityItem, pillars map[string]config.PillarConfig) []PillarMatch {
	matches := make([]PillarMatch, 0)
	seen := map[string]bool{}

	labelsLower := make([]string, 0, len(item.Labels))
	for _, label := range item.Labels {
		labelsLower = append(labelsLower, strings.ToLower(label))
	}

	for key, pillar := range pillars {
		for _, label := range pillar.Labels {
			for _, itemLabel := range labelsLower {
				if strings.EqualFold(label, itemLabel) {
					addMatch(&matches, seen, key, "high", "label: "+label, item)
					break
				}
			}
		}
	}

	for key, pillar := range pillars {
		for _, repo := range pillar.Repos {
			if strings.EqualFold(repo, item.Repo) {
				addMatch(&matches, seen, key, "high", "repo: "+repo, item)
			}
		}
	}

	for key, pillar := range pillars {
		patterns := pillar.Paths
		if len(patterns) == 0 {
			patterns = defaultPathPatterns(key)
		}
		for _, pattern := range patterns {
			if matchAnyPath(item.Files, pattern) {
				addMatch(&matches, seen, key, "medium", "path: "+pattern, item)
			}
		}
	}

	combinedText := strings.ToLower(item.Title + " " + item.Body)
	for key, pillar := range pillars {
		for _, signal := range pillar.Signals {
			if signal == "" {
				continue
			}
			if strings.Contains(combinedText, strings.ToLower(signal)) {
				addMatch(&matches, seen, key, "low", "keyword: "+signal, item)
			}
		}
	}

	return matches
}

func addMatch(matches *[]PillarMatch, seen map[string]bool, pillar, confidence, reason string, item ActivityItem) {
	if seen[pillar] {
		return
	}
	seen[pillar] = true
	*matches = append(*matches, PillarMatch{Pillar: pillar, Confidence: confidence, Reason: reason, Item: item})
}

func matchAnyPath(paths []string, pattern string) bool {
	for _, path := range paths {
		matched, _ := filepath.Match(pattern, path)
		if matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func defaultPathPatterns(pillar string) []string {
	switch strings.ToLower(pillar) {
	case "reliability":
		return []string{"tests/**", "test/**", "**/*_test.go", "**/*.spec.*"}
	case "developer-experience":
		return []string{"docs/**", "doc/**", "**/*.md"}
	default:
		return nil
	}
}
