package tme

import (
	"context"
	"fmt"
)

func (c *Client) GetArtistDetail(ctx context.Context, artistID string) (*Artist, error) {
	mid := extractMID(artistID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musichallSinger.SingerInfoInter",
			Method: "GetSingerDetail",
			Param: map[string]any{
				"singer_mid": mid,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("artist detail failed: code=%d", sub.Code)
	}

	info := sub.Data
	if d, ok := sub.Data["singerinfo"].(map[string]any); ok {
		info = d
	}
	if d, ok := sub.Data["data"].(map[string]any); ok {
		info = d
	}

	artist := &Artist{
		ID:          "qqmusic:artist:" + mid,
		Name:        getStringField(info, "name", "singer_name"),
		AvatarURL:   buildArtworkURL(getString(info, "pic")),
		Description: getString(info, "desc"),
		SongCount:   getInt(info, "song_num"),
		AlbumCount:  getInt(info, "album_num"),
	}

	if alias := getString(info, "alias"); alias != "" {
		artist.Alias = []string{alias}
	}

	return artist, nil
}

func (c *Client) GetArtistTracks(ctx context.Context, artistID string, limit, page int) ([]Song, error) {
	mid := extractMID(artistID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "musichall.song_list_server",
			Method: "GetSingerSongList",
			Param: map[string]any{
				"singer_mid": mid,
				"num":        limit,
				"page":       page,
				"order":      "listen",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("artist tracks failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}

func (c *Client) GetAlbumDetail(ctx context.Context, albumID string) (*Album, error) {
	mid := extractMID(albumID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musichallAlbum.AlbumInfoServer",
			Method: "GetAlbumDetail",
			Param: map[string]any{
				"album_mid": mid,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("album detail failed: code=%d", sub.Code)
	}

	info := sub.Data
	if d, ok := sub.Data["data"].(map[string]any); ok {
		info = d
	}

	album := &Album{
		ID:          "qqmusic:album:" + mid,
		Title:       getStringField(info, "name", "album_name"),
		ReleaseDate: getString(info, "release_date"),
		SongCount:   getInt(info, "songnum"),
		Description: getString(info, "desc"),
		ArtworkURL:  buildArtworkURL(getString(info, "pmid")),
	}

	for _, singer := range getMapSlice(info, "singer_list") {
		if name := getStringField(singer, "name"); name != "" {
			album.Artists = append(album.Artists, name)
		}
	}

	return album, nil
}

func (c *Client) GetAlbumTracks(ctx context.Context, albumID string, limit, page int) ([]Song, error) {
	mid := extractMID(albumID)

	resp, err := c.Call(ctx, map[string]MusicuSubRequest{
		"req_0": {
			Module: "music.musichallAlbum.AlbumSongList",
			Method: "GetAlbumSongList",
			Param: map[string]any{
				"album_mid": mid,
				"num":       limit,
				"page":      page,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	sub, ok := resp.Req["req_0"]
	if !ok || sub.Code != 0 {
		return nil, fmt.Errorf("album tracks failed: code=%d", sub.Code)
	}

	return c.parseSongs(sub.Data), nil
}
