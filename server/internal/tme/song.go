package tme

import (
	"context"
	"fmt"
)

func (c *Client) GetSongDetail(ctx context.Context, songID string) (*SongDetail, error) {
	mid := extractMID(songID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.trackInfo.UniformRuleCtrl",
			Method: "CgiGetTrackInfo",
			Param: map[string]any{
				"ids":     []string{mid},
				"types":   []int{0},
				"modules": []string{"basic"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("song detail failed: code=%d", sub.Code)
	}

	tracks := getMapSlice(sub.Data, "tracks")
	if len(tracks) == 0 {
		return nil, fmt.Errorf("song not found: %s", songID)
	}

	track := tracks[0]
	detail := &SongDetail{
		Song: Song{
			ID:              "qqmusic:" + getStringField(track, "mid", "songmid"),
			Title:           getStringField(track, "name", "title", "songname"),
			DurationSeconds: getInt(track, "interval"),
		},
	}

	for _, singer := range getMapSlice(track, "singer") {
		if name := getString(singer, "name"); name != "" {
			detail.Artists = append(detail.Artists, name)
		}
	}

	if album, ok := track["album"]; ok {
		if m, ok := album.(map[string]any); ok {
			detail.Album = getStringField(m, "name", "title")
			detail.ArtworkURL = buildArtworkURL(getString(m, "pmid"))
		}
	}

	return detail, nil
}

func (c *Client) GetSongURL(ctx context.Context, songID string) (*SongURL, error) {
	mid := extractMID(songID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.vkey.GetVkey",
			Method: "UrlGetVkey",
			Param: map[string]any{
				"guid":      "10000",
				"songmid":   []string{mid},
				"songtype":  []int{0},
				"uin":       "0",
				"loginflag": 1,
				"platform":  "20",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("song url failed: code=%d", sub.Code)
	}

	urls := getMapSlice(sub.Data, "midurlinfo")
	if len(urls) == 0 {
		return &SongURL{SongID: songID}, nil
	}

	item := urls[0]
	purl := getString(item, "purl")
	if purl == "" {
		return &SongURL{SongID: songID}, nil
	}

	fullURL := purl
	if !hasPrefix(purl, "http") {
		fullURL = streamDomain + purl
	}

	return &SongURL{
		SongID:           songID,
		URL:              fullURL,
		ExpiresInSeconds: getInt(sub.Data, "expiration"),
	}, nil
}

func (c *Client) GetSimilarSongs(ctx context.Context, songID string, limit int) ([]Song, error) {
	mid := extractMID(songID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.recommend.TrackRelationServer",
			Method: "GetSimilarSongs",
			Param: map[string]any{
				"songid": mid,
				"num":    limit,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("similar songs failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}

func (c *Client) GetRelatedPlaylists(ctx context.Context, songID string, limit int) ([]Playlist, error) {
	mid := extractMID(songID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.recommend.TrackRelationServer",
			Method: "GetRelatedPlaylist",
			Param: map[string]any{
				"songid": mid,
				"num":    limit,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("related playlists failed: code=%d", sub.Code)
	}

	var playlists []Playlist
	for _, item := range getMapSlice(sub.Data, "vecPlaylist") {
		playlists = append(playlists, Playlist{
			ID:         "qqmusic:playlist:" + getString(item, "tid"),
			Name:       getString(item, "title"),
			SongCount:  getInt(item, "song_num"),
			ArtworkURL: buildArtworkURL(getString(item, "cover")),
		})
	}
	return playlists, nil
}

func extractMID(id string) string {
	prefixes := []string{"qqmusic:", "qqmusic:track:", "qqmusic:song:"}
	for _, p := range prefixes {
		if len(id) > len(p) && id[:len(p)] == p {
			return id[len(p):]
		}
	}
	return id
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
