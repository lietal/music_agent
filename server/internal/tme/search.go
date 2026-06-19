package tme

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
)

func (c *Client) SearchSongs(ctx context.Context, keyword string, limit int) ([]Song, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.search.SearchCgiService",
			Method: "DoSearchForQQMusicMobile",
			Param: map[string]any{
				"searchid":     strconv.FormatInt(rand.Int63(), 10),
				"query":        keyword,
				"search_type":  0,
				"num_per_page": limit,
				"page_num":     1,
				"highlight":    1,
				"grp":          1,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("search failed: code=%d", sub.Code)
	}

	songs := c.parseSongs(sub.Data)
	return songs, nil
}

func (c *Client) SearchArtists(ctx context.Context, keyword string, limit, page int) ([]Artist, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musichallSinger.SingerList",
			Method: "GetSingerList",
			Param: map[string]any{
				"searchid":   strconv.FormatInt(rand.Int63(), 10),
				"query":      keyword,
				"num_per_page": limit,
				"page_num":     page,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("artist search failed: code=%d", sub.Code)
	}

	var artists []Artist
	for _, item := range getNestedMapSlice(sub.Data, "body", "singerlist") {
		artists = append(artists, Artist{
			ID:        "qqmusic:artist:" + getStringField(item, "mid", "singer_mid"),
			Name:      getStringField(item, "name", "singer_name"),
			AvatarURL: buildArtworkURL(getString(item, "pic")),
			SongCount: getInt(item, "songs"),
		})
	}
	return artists, nil
}

func (c *Client) SearchAlbums(ctx context.Context, keyword string, limit, page int) ([]Album, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musichallAlbum.AlbumListServer",
			Method: "GetAlbumList",
			Param: map[string]any{
				"searchid":   strconv.FormatInt(rand.Int63(), 10),
				"query":      keyword,
				"num_per_page": limit,
				"page_num":     page,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("album search failed: code=%d", sub.Code)
	}

	var albums []Album
	for _, item := range getNestedMapSlice(sub.Data, "body", "albumlist") {
		albums = append(albums, Album{
			ID:         "qqmusic:album:" + getStringField(item, "mid", "album_mid"),
			Title:      getStringField(item, "name", "album_name"),
			Artists:    getStringSlice(item, "singer_list"),
			ArtworkURL: buildArtworkURL(getString(item, "pic")),
			SongCount:  getInt(item, "song_count"),
		})
	}
	return albums, nil
}

func (c *Client) SearchPlaylists(ctx context.Context, keyword string, limit, page int) ([]Playlist, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.playlist.PlaylistSquare",
			Method: "GetPlaylistByTag",
			Param: map[string]any{
				"searchid":   strconv.FormatInt(rand.Int63(), 10),
				"query":      keyword,
				"num_per_page": limit,
				"page_num":     page,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("playlist search failed: code=%d", sub.Code)
	}

	var playlists []Playlist
	for _, item := range getNestedMapSlice(sub.Data, "body", "playlists") {
		playlists = append(playlists, Playlist{
			ID:         "qqmusic:playlist:" + getStringField(item, "tid", "dissid"),
			Name:       getStringField(item, "title", "dissname"),
			SongCount:  getInt(item, "song_num"),
			ArtworkURL: buildArtworkURL(getString(item, "pic")),
		})
	}
	return playlists, nil
}

func (c *Client) parseSongs(data map[string]any) []Song {
	var items []map[string]any

	if body, ok := data["body"].(map[string]any); ok {
		items = getMapSlice(body, "item_song")
	}
	if items == nil {
		items = getMapSlice(data, "songlist")
	}
	if items == nil {
		items = getMapSlice(data, "list")
	}
	if items == nil {
		items = getMapSlice(data, "songs")
	}

	var songs []Song
	for _, item := range items {
		song := Song{
			ID:              "qqmusic:" + getStringField(item, "mid", "songmid", "songMid"),
			Title:           getStringField(item, "name", "title", "songname", "songName"),
			DurationSeconds: getInt(item, "interval"),
		}

		for _, singer := range getMapSlice(item, "singer") {
			if name := getStringField(singer, "name"); name != "" {
				song.Artists = append(song.Artists, name)
			}
		}
		if len(song.Artists) == 0 {
			if s := getString(item, "singer_name"); s != "" {
				song.Artists = []string{s}
			}
			if s := getString(item, "singerName"); s != "" {
				song.Artists = []string{s}
			}
		}

		if album, ok := item["album"]; ok {
			if m, ok := album.(map[string]any); ok {
				song.Album = getStringField(m, "name", "title")
				song.ArtworkURL = buildArtworkURL(getString(m, "pmid"))
			}
		}
		if song.Album == "" {
			song.Album = getStringField(item, "albumname", "albumName", "album_name")
		}
		if song.ArtworkURL == "" {
			if albumMid := getStringField(item, "albummid", "albumMid"); albumMid != "" {
				song.ArtworkURL = fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R300x300M000%s.jpg", albumMid)
			}
		}

		songs = append(songs, song)
	}
	return songs
}
