package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type mockSearchSongs struct{}

func NewMockSearchSongs() Tool {
	return &mockSearchSongs{}
}

func (m *mockSearchSongs) Name() string        { return "search_songs" }
func (m *mockSearchSongs) Description() string { return "Searches for songs matching a query" }

func (m *mockSearchSongs) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	query := ""
	if q, ok := args["query"]; ok {
		query = fmt.Sprintf("%v", q)
	}

	songs := []map[string]any{
		{"id": "song-1", "title": "晴天", "artist": "周杰伦", "year": 2003},
		{"id": "song-2", "title": "七里香", "artist": "周杰伦", "year": 2004},
		{"id": "song-3", "title": "夜曲", "artist": "周杰伦", "year": 2005},
		{"id": "song-4", "title": "青花瓷", "artist": "周杰伦", "year": 2007},
		{"id": "song-5", "title": "稻香", "artist": "周杰伦", "year": 2008},
	}

	if !strings.Contains(query, "周杰伦") && !strings.Contains(query, "jay") {
		songs = []map[string]any{
			{"id": "song-1", "title": "Bohemian Rhapsody", "artist": "Queen", "year": 1975},
			{"id": "song-2", "title": "Stairway to Heaven", "artist": "Led Zeppelin", "year": 1971},
			{"id": "song-3", "title": "Hotel California", "artist": "Eagles", "year": 1977},
		}
	}

	data, err := json.Marshal(songs)
	if err != nil {
		return ToolResult{Error: err.Error()}, nil
	}

	return ToolResult{Data: string(data)}, nil
}

type mockRecommendSongs struct{}

func NewMockRecommendSongs() Tool {
	return &mockRecommendSongs{}
}

func (m *mockRecommendSongs) Name() string        { return "recommend_songs" }
func (m *mockRecommendSongs) Description() string { return "Recommends songs similar to a given song" }

func (m *mockRecommendSongs) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	recs := []map[string]any{
		{"id": "rec-1", "title": "Killer Queen", "artist": "Queen", "score": 0.95},
		{"id": "rec-2", "title": "Don't Stop Me Now", "artist": "Queen", "score": 0.91},
		{"id": "rec-3", "title": "Somebody to Love", "artist": "Queen", "score": 0.88},
	}

	data, err := json.Marshal(recs)
	if err != nil {
		return ToolResult{Error: err.Error()}, nil
	}

	return ToolResult{Data: string(data)}, nil
}
