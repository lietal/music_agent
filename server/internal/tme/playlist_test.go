package tme

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPlaylistDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"dirinfo": map[string]any{
						"title":    "周杰伦精选",
						"song_num": float64(50),
						"cover":    "000playlist",
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	pl, err := c.GetPlaylistDetail(t.Context(), "qqmusic:playlist:123")
	if err != nil {
		t.Fatal(err)
	}
	if pl.Name != "周杰伦精选" {
		t.Errorf("got %s", pl.Name)
	}
	if pl.SongCount != 50 {
		t.Errorf("got %d", pl.SongCount)
	}
}

func TestGetPlaylistSongs(t *testing.T) {
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
								"mid":  "pl001",
								"name": "Playlist Song",
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

	songs, err := c.GetPlaylistSongs(t.Context(), "qqmusic:playlist:123", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 1 {
		t.Fatalf("expected 1, got %d", len(songs))
	}
}

func TestSearchArtists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"singerlist": []any{
							map[string]any{
								"mid":         "artist1",
								"singer_name": "周杰伦",
								"pic":         "000artist",
								"songs":       float64(200),
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

	artists, err := c.SearchArtists(t.Context(), "周杰伦", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 {
		t.Fatalf("expected 1, got %d", len(artists))
	}
	if artists[0].Name != "周杰伦" {
		t.Errorf("got %s", artists[0].Name)
	}
}

func TestSearchAlbums(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"albumlist": []any{
							map[string]any{
								"album_mid":  "alb1",
								"album_name": "叶惠美",
								"pic":        "000album",
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

	albums, err := c.SearchAlbums(t.Context(), "叶惠美", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(albums) != 1 {
		t.Fatalf("expected 1, got %d", len(albums))
	}
}

func TestSearchPlaylists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"playlists": []any{
							map[string]any{
								"tid":      "pl1",
								"dissname": "周杰伦歌单",
								"song_num": float64(30),
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

	playlists, err := c.SearchPlaylists(t.Context(), "周杰伦", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) != 1 {
		t.Fatalf("expected 1, got %d", len(playlists))
	}
}

func TestGetTrackComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"commentlist": []any{
						map[string]any{
							"commentid":   "c1",
							"nick":        "User",
							"rootcomment": "Nice!",
							"praisenum":   float64(5),
							"time":        float64(1718000000),
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	comments, err := c.GetTrackComments(t.Context(), "qqmusic:001abc", "new", 5, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1, got %d", len(comments))
	}
}

func TestGetRecommendPlaylists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"vecPlaylist": []any{
						map[string]any{
							"tid":      "rec1",
							"title":    "推荐歌单",
							"song_num": float64(20),
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	playlists, err := c.GetRecommendPlaylists(t.Context(), 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) != 1 {
		t.Fatalf("expected 1, got %d", len(playlists))
	}
}
