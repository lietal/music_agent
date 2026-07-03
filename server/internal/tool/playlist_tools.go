package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

type playlistTools struct {
	db *pgxpool.Pool
}

func NewPlaylistTools(db *pgxpool.Pool) map[string]Tool {
	pt := &playlistTools{db: db}
	return map[string]Tool{
		"create_playlist":   &createPlaylistTool{pt},
		"add_to_playlist":   &addToPlaylistTool{pt},
		"remove_song":       &removeSongTool{pt},
		"list_playlists":    &listPlaylistsTool{pt},
		"get_playlist":      &getPlaylistTool{pt},
		"rename_playlist":   &renamePlaylistTool{pt},
		"delete_playlist":   &deletePlaylistTool{pt},
		"play_playlist":     &playPlaylistTool{pt},
	}
}

func (pt *playlistTools) userID(ctx context.Context) string {
	return UserIDFromContext(ctx)
}

type createPlaylistTool struct{ pt *playlistTools }

func (t *createPlaylistTool) Name() string       { return "create_playlist" }
func (t *createPlaylistTool) Description() string {
	return "Create a new empty playlist or 歌单. Use when user wants to create/make a new playlist. Args: {\"name\": \"playlist name\"}"
}

func (t *createPlaylistTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	name := getStringArg(args, "name")
	if name == "" {
		return ToolResult{Error: "name is required"}, nil
	}
	uid := t.pt.userID(ctx)
	if uid == "" {
		return ToolResult{Error: "not authenticated"}, nil
	}
	id := uuid.New().String()
	_, err := t.pt.db.Exec(ctx,
		`INSERT INTO playlists (id, user_id, name) VALUES ($1,$2,$3)`, id, uid, name)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("create failed: %v", err)}, nil
	}
	data, _ := json.Marshal(map[string]string{"id": id, "name": name})
	return ToolResult{Data: string(data)}, nil
}

// ── add_to_playlist ──

type addToPlaylistTool struct{ pt *playlistTools }

func (t *addToPlaylistTool) Name() string       { return "add_to_playlist" }
func (t *addToPlaylistTool) Description() string {
	return "Add a song to a playlist by playlist name. Use this when user wants to save/add/collect a song into a playlist or 歌单. " +
		"Args: {\"playlist\": \"playlist name\", \"songId\": \"qqmusic:...\", \"title\": \"song title\", \"artist\": \"artist name\"}"
}

func (t *addToPlaylistTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	playlistName := getStringArg(args, "playlist")
	songID := getStringArg(args, "songId")
	title := getStringArg(args, "title")
	artist := getStringArg(args, "artist")
	if playlistName == "" || songID == "" {
		return ToolResult{Error: "playlist name and songId are required"}, nil
	}
	uid := t.pt.userID(ctx)
	if uid == "" {
		return ToolResult{Error: "not authenticated"}, nil
	}
	coverURL := getStringArg(args, "coverUrl")
	var pid string
	err := t.pt.db.QueryRow(ctx,
		`SELECT id FROM playlists WHERE user_id=$1 AND name=$2 LIMIT 1`, uid, playlistName).Scan(&pid)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("playlist not found: %s", playlistName)}, nil
	}
	_, err = t.pt.db.Exec(ctx,
		`INSERT INTO playlist_songs (playlist_id, song_id, title, artist, cover_url) VALUES ($1,$2,$3,$4,$5)`,
		pid, songID, title, artist, coverURL)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("add failed: %v", err)}, nil
	}
	_, execErr := t.pt.db.Exec(ctx, `UPDATE playlists SET updated_at=now() WHERE id=$1`, pid)
	if execErr != nil {
		slog.Warn("failed to update playlist timestamp", "error", execErr, "playlistID", pid)
	}
	return ToolResult{Data: "ok"}, nil
}

// ── remove_song ──

type removeSongTool struct{ pt *playlistTools }

func (t *removeSongTool) Name() string        { return "remove_song" }
func (t *removeSongTool) Description() string  { return "Remove a song from a playlist by name. Args: {\"playlist\": \"playlist name\", \"songId\": \"...\"}" }

func (t *removeSongTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	playlistName := getStringArg(args, "playlist")
	songID := getStringArg(args, "songId")
	if playlistName == "" || songID == "" {
		return ToolResult{Error: "playlist name and songId are required"}, nil
	}
	uid := t.pt.userID(ctx)
	if uid == "" {
		return ToolResult{Error: "not authenticated"}, nil
	}
	var pid string
	err := t.pt.db.QueryRow(ctx,
		`SELECT id FROM playlists WHERE user_id=$1 AND name=$2 LIMIT 1`, uid, playlistName).Scan(&pid)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("playlist not found: %s", playlistName)}, nil
	}
	_, err = t.pt.db.Exec(ctx,
		`DELETE FROM playlist_songs WHERE playlist_id=$1 AND song_id=$2`, pid, songID)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("remove failed: %v", err)}, nil
	}
	return ToolResult{Data: "ok"}, nil
}

// ── list_playlists ──

type listPlaylistsTool struct{ pt *playlistTools }

func (t *listPlaylistsTool) Name() string       { return "list_playlists" }
func (t *listPlaylistsTool) Description() string {
	return "List all playlists/歌单 for the current user. Use when user asks about their playlists, 歌单列表, or 我的歌单. Args: {}"
}

func (t *listPlaylistsTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	uid := t.pt.userID(ctx)
	if uid == "" {
		return ToolResult{Error: "not authenticated"}, nil
	}
	rows, err := t.pt.db.Query(ctx,
		`SELECT id, name FROM playlists WHERE user_id=$1 ORDER BY updated_at DESC`, uid)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("list failed: %v", err)}, nil
	}
	defer rows.Close()
	var result []map[string]string
	for rows.Next() {
		var id, name string
		if rows.Scan(&id, &name) == nil {
			result = append(result, map[string]string{"id": id, "name": name})
		}
	}
	if result == nil {
		result = []map[string]string{}
	}
	data, _ := json.Marshal(result)
	return ToolResult{Data: string(data)}, nil
}

// ── get_playlist ──

type getPlaylistTool struct{ pt *playlistTools }

func (t *getPlaylistTool) Name() string        { return "get_playlist" }
func (t *getPlaylistTool) Description() string  { return "Get songs in a playlist by name. Args: {\"playlist\": \"playlist name\"}" }

func (t *getPlaylistTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	playlistName := getStringArg(args, "playlist")
	if playlistName == "" {
		return ToolResult{Error: "playlist name is required"}, nil
	}
	uid := t.pt.userID(ctx)
	if uid == "" {
		return ToolResult{Error: "not authenticated"}, nil
	}
	var pid string
	err := t.pt.db.QueryRow(ctx,
		`SELECT id FROM playlists WHERE user_id=$1 AND name=$2 LIMIT 1`, uid, playlistName).Scan(&pid)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("playlist not found: %s", playlistName)}, nil
	}
	rows, err := t.pt.db.Query(ctx,
		`SELECT song_id, title, artist, cover_url FROM playlist_songs WHERE playlist_id=$1 ORDER BY added_at`, pid)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("get failed: %v", err)}, nil
	}
	defer rows.Close()
	var songs []map[string]string
	for rows.Next() {
		var sid, t, a, c string
		if rows.Scan(&sid, &t, &a, &c) == nil {
			songs = append(songs, map[string]string{"songId": sid, "title": t, "artist": a, "coverUrl": c})
		}
	}
	if songs == nil {
		songs = []map[string]string{}
	}
	data, _ := json.Marshal(songs)
	return ToolResult{Data: string(data)}, nil
}

type renamePlaylistTool struct{ pt *playlistTools }

func (t *renamePlaylistTool) Name() string       { return "rename_playlist" }
func (t *renamePlaylistTool) Description() string { return "Rename a playlist. Args: {\"playlist\": \"current name\", \"newName\": \"new name\"}" }

func (t *renamePlaylistTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	oldName := getStringArg(args, "playlist")
	newName := getStringArg(args, "newName")
	if oldName == "" || newName == "" {
		return ToolResult{Error: "playlist name and newName are required"}, nil
	}
	uid := t.pt.userID(ctx)
	if uid == "" {
		return ToolResult{Error: "not authenticated"}, nil
	}
	_, err := t.pt.db.Exec(ctx,
		`UPDATE playlists SET name=$1, updated_at=now() WHERE user_id=$2 AND name=$3`,
		newName, uid, oldName)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("rename failed: %v", err)}, nil
	}
	return ToolResult{Data: fmt.Sprintf("歌单已改名为 %s", newName)}, nil
}

func getStringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

type deletePlaylistTool struct{ pt *playlistTools }

func (t *deletePlaylistTool) Name() string { return "delete_playlist" }
func (t *deletePlaylistTool) Description() string {
	return "Delete a playlist by name. Use when user wants to delete/remove a playlist or 歌单. Args: {\"playlist\": \"playlist name\"}"
}

func (t *deletePlaylistTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	playlistName := getStringArg(args, "playlist")
	if playlistName == "" {
		return ToolResult{Error: "playlist name is required"}, nil
	}
	uid := t.pt.userID(ctx)
	_, err := t.pt.db.Exec(ctx,
		`DELETE FROM playlists WHERE user_id=$1 AND name=$2`, uid, playlistName)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("delete failed: %v", err)}, nil
	}
	return ToolResult{Data: fmt.Sprintf("歌单 %s 已删除", playlistName)}, nil
}
