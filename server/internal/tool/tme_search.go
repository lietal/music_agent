package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/music-agent/music-agent/internal/tme"
)

type TMESearchSongs struct {
	client *tme.Client
}

func NewTMESearchSongs() *TMESearchSongs {
	return &TMESearchSongs{
		client: tme.NewClient(),
	}
}

func (t *TMESearchSongs) Name() string {
	return "search_songs"
}

func (t *TMESearchSongs) Description() string {
	return "Search for songs on QQ Music by keyword. Returns song id, title, artists, album, duration, and artwork URL."
}

func (t *TMESearchSongs) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	keyword := ""
	if kw, ok := args["keyword"]; ok {
		keyword = fmt.Sprintf("%v", kw)
	}
	if q, ok := args["query"]; ok && keyword == "" {
		keyword = fmt.Sprintf("%v", q)
	}
	if keyword == "" {
		return ToolResult{}, fmt.Errorf("keyword is required")
	}

	limit := 5
	if l, ok := args["limit"]; ok {
		switch v := l.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		}
	}

	songs, err := t.client.SearchSongs(ctx, keyword, limit)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}

	data, err := json.Marshal(songs)
	if err != nil {
		return ToolResult{Error: "marshal failed"}, nil
	}

	return ToolResult{Data: string(data)}, nil
}

func (t *TMESearchSongs) IsAvailable(ctx context.Context) bool {
	_, err := t.client.SearchSongs(ctx, "test", 1)
	return err == nil
}
