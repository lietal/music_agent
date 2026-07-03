package agent

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/music-agent/music-agent/internal/tool"
)

type rankedSong struct {
	Data  map[string]any
	Score float64
}

func OrchestrateRecommendation(query string, results []tool.Observation) []map[string]any {
	if len(results) == 0 {
		return nil
	}
	var allSongs []map[string]any
	for _, obs := range results {
		if obs.Result.Data != "" {
			var parsed any
			json.Unmarshal([]byte(obs.Result.Data), &parsed)
			if arr, ok := parsed.([]any); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						allSongs = append(allSongs, m)
					}
				}
			}
		}
	}
	if len(allSongs) == 0 {
		return nil
	}
	var ranked []rankedSong
	for _, s := range allSongs {
		score := 0.5
		title := getString(s, "title")
		artist := getString(s, "artist")
		if strings.Contains(title, query) || strings.Contains(artist, query) {
			score += 0.3
		}
		if strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
			score += 0.2
		}
		ranked = append(ranked, rankedSong{Data: s, Score: score})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	result := make([]map[string]any, len(ranked))
	for i, r := range ranked {
		result[i] = r.Data
	}
	return result
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
