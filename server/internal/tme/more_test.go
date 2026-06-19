package tme

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRecommendTracks(t *testing.T) {
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
								"mid":  "rec001",
								"name": "推荐曲目",
								"singer": []any{
									map[string]any{"name": "歌手"},
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

	songs, err := c.GetRecommendTracks(t.Context(), "guess", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 1 {
		t.Fatalf("expected 1, got %d", len(songs))
	}
}

func TestGetRecommendTracks_Radar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"body": map[string]any{
						"item_song": []any{},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	_, err := c.GetRecommendTracks(t.Context(), "radar", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetChartDetail(t *testing.T) {
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
								"mid":      "chart001",
								"name":     "榜单歌曲",
								"interval": float64(200),
								"singer":   []any{map[string]any{"name": "歌手"}},
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

	songs, err := c.GetChartDetail(t.Context(), "qqmusic:chart:26", 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) == 0 {
		t.Error("expected songs")
	}
}

func TestGetArtistDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"singerinfo": map[string]any{
						"name":      "周杰伦",
						"pic":       "000artist",
						"desc":      "华语流行歌手",
						"song_num":  float64(200),
						"album_num": float64(15),
						"alias":     "Jay Chou",
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	artist, err := c.GetArtistDetail(t.Context(), "qqmusic:artist:002abc")
	if err != nil {
		t.Fatal(err)
	}
	if artist.Name != "周杰伦" {
		t.Errorf("got %s", artist.Name)
	}
	if artist.SongCount != 200 {
		t.Errorf("got %d", artist.SongCount)
	}
	if len(artist.Alias) == 0 || artist.Alias[0] != "Jay Chou" {
		t.Errorf("got %v", artist.Alias)
	}
}

func TestGetAlbumDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"data": map[string]any{
						"name":         "叶惠美",
						"release_date": "2003-07-31",
						"songnum":      float64(10),
						"desc":         "经典专辑",
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	album, err := c.GetAlbumDetail(t.Context(), "qqmusic:album:003abc")
	if err != nil {
		t.Fatal(err)
	}
	if album.Title != "叶惠美" {
		t.Errorf("got %s", album.Title)
	}
	if album.SongCount != 10 {
		t.Errorf("got %d", album.SongCount)
	}
}
