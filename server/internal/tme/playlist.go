package tme

import (
	"context"
	"fmt"
)

func (c *Client) GetUserPlaylists(ctx context.Context) ([]Playlist, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musicasset.PlaylistBaseRead",
			Method: "GetPlaylistByUin",
			Param: map[string]any{
				"uin": c.comm.QQ,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("user playlists failed: code=%d", sub.Code)
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

func (c *Client) GetPlaylistDetail(ctx context.Context, playlistID string) (*Playlist, error) {
	mid := extractMID(playlistID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.songlist.SonglistRead",
			Method: "GetDetail",
			Param: map[string]any{
				"disstid": mid,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("playlist detail failed: code=%d", sub.Code)
	}

	dirinfo := sub.Data
	if d, ok := sub.Data["dirinfo"].(map[string]any); ok {
		dirinfo = d
	}

	return &Playlist{
		ID:         "qqmusic:playlist:" + mid,
		Name:       getString(dirinfo, "title"),
		SongCount:  getInt(dirinfo, "song_num"),
		ArtworkURL: buildArtworkURL(getString(dirinfo, "cover")),
	}, nil
}

func (c *Client) GetPlaylistSongs(ctx context.Context, playlistID string, limit, page int) ([]Song, error) {
	mid := extractMID(playlistID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.songlist.SonglistRead",
			Method: "GetDetail",
			Param: map[string]any{
				"disstid":      mid,
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
		return nil, fmt.Errorf("playlist songs failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}

func (c *Client) GetRecommendPlaylists(ctx context.Context, limit, page int) ([]Playlist, error) {
	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.playlist.PlaylistSquare",
			Method: "GetRecommendFeed",
			Param: map[string]any{
				"num":  limit,
				"page": page,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("recommend playlists failed: code=%d", sub.Code)
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
