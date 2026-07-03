package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

type playPlaylistTool struct{ pt *playlistTools }

func (t *playPlaylistTool) Name() string { return "play_playlist" }

func (t *playPlaylistTool) Description() string {
	return "Play songs from a playlist by name. Use when user wants to play or listen to a playlist. Args: {\"playlist\": \"playlist name\"}"
}

func (t *playPlaylistTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	playlistName := getStringArg(args, "playlist")
	if playlistName == "" {
		return ToolResult{Error: "playlist name is required"}, nil
	}

	userID := t.pt.userID(ctx)
	rows, err := t.pt.db.Query(ctx,
		`SELECT ps.song_id, ps.title, ps.artist, ps.cover_url FROM playlist_songs ps
		 JOIN playlists p ON ps.playlist_id = p.id
		 WHERE p.name = $1 AND p.user_id = $2`, playlistName, userID)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("query failed: %v", err)}, nil
	}
	defer rows.Close()

	type songEntry struct {
		ID              string `json:"id"`
		Title           string `json:"title"`
		Artist          string `json:"artist"`
		CoverURL        string `json:"coverUrl"`
		DurationSeconds int    `json:"durationSeconds"`
	}

	var songs []songEntry
	for rows.Next() {
		var s songEntry
		if err := rows.Scan(&s.ID, &s.Title, &s.Artist, &s.CoverURL); err != nil {
			continue
		}
		songs = append(songs, s)
	}

	if songs == nil {
		songs = []songEntry{}
	}

	data, _ := json.Marshal(map[string]any{"songs": songs})
	return ToolResult{Data: string(data)}, nil
}
