package tme

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLyrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"lyric": "74657374",
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	lyrics, err := c.GetLyrics(t.Context(), "qqmusic:001abc")
	if err != nil {
		t.Fatal(err)
	}
	if lyrics.SongID != "qqmusic:001abc" {
		t.Errorf("got %s", lyrics.SongID)
	}
}

func TestGetSimilarSongs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"item_song": []any{
							map[string]any{
								"mid":      "sim001",
								"name":     "Similar Song",
								"interval": float64(200),
								"singer":   []any{map[string]any{"name": "Singer"}},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	songs, err := c.GetSimilarSongs(t.Context(), "qqmusic:001abc", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) == 0 {
		t.Error("expected songs")
	}
}

func TestGetRelatedPlaylists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"vecPlaylist": []any{
						map[string]any{
							"tid":      "rel1",
							"title":    "Related Playlist",
							"song_num": float64(10),
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	playlists, err := c.GetRelatedPlaylists(t.Context(), "qqmusic:001abc", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) == 0 {
		t.Error("expected playlists")
	}
}

func TestGetUserPlaylists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"vecPlaylist": []any{
						map[string]any{
							"tid":      "user1",
							"title":    "My Playlist",
							"song_num": float64(5),
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	playlists, err := c.GetUserPlaylists(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) == 0 {
		t.Error("expected playlists")
	}
}

func TestGetArtistTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"item_song": []any{
							map[string]any{
								"mid":  "art001",
								"name": "Artist Track",
								"singer": []any{
									map[string]any{"name": "Artist"},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	songs, err := c.GetArtistTracks(t.Context(), "qqmusic:artist:002", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) == 0 {
		t.Error("expected songs")
	}
}

func TestGetAlbumTracks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"item_song": []any{
							map[string]any{
								"mid":  "alb001",
								"name": "Album Track",
								"singer": []any{
									map[string]any{"name": "Singer"},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	songs, err := c.GetAlbumTracks(t.Context(), "qqmusic:album:003", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) == 0 {
		t.Error("expected songs")
	}
}
