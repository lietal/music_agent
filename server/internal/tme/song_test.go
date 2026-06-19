package tme

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSongDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"tracks": []any{
						map[string]any{
							"mid":      "0039MnYb0qxYhV",
							"name":     "晴天",
							"interval": float64(269),
							"singer":   []any{map[string]any{"name": "周杰伦"}},
							"album":    map[string]any{"name": "叶惠美", "pmid": "000MkMni19ClKG_5"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	detail, err := c.GetSongDetail(t.Context(), "qqmusic:0039MnYb0qxYhV")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Title != "晴天" {
		t.Errorf("got %s", detail.Title)
	}
	if detail.Album != "叶惠美" {
		t.Errorf("got %s", detail.Album)
	}
}

func TestGetSongDetail_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"tracks": []any{},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	_, err := c.GetSongDetail(t.Context(), "missing")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestGetSongURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"midurlinfo": []any{
						map[string]any{
							"purl":   "C400001abc.m4a",
							"result": float64(0),
						},
					},
					"expiration": float64(3600),
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	url, err := c.GetSongURL(t.Context(), "qqmusic:001abc")
	if err != nil {
		t.Fatal(err)
	}
	if url.URL == "" {
		t.Error("expected non-empty URL")
	}
	if url.ExpiresInSeconds != 3600 {
		t.Errorf("expected 3600, got %d", url.ExpiresInSeconds)
	}
}

func TestGetSongURL_EmptyPurL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"midurlinfo": []any{},
				},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	url, err := c.GetSongURL(t.Context(), "001abc")
	if err != nil {
		t.Fatal(err)
	}
	if url.URL != "" {
		t.Errorf("expected empty URL, got %s", url.URL)
	}
}

func TestStripLRCTimestamps(t *testing.T) {
	input := "[00:15.50]雨下整夜\n[00:20.00]我的爱溢出就像雨水"
	got := stripLRCTimestamps(input)
	if got != "雨下整夜\n我的爱溢出就像雨水" {
		t.Errorf("got %q", got)
	}
}

func TestGetChartCategories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"group": []any{
						map[string]any{
							"toplist": []any{
								map[string]any{
									"topId":            "26",
									"title":            "热歌榜",
									"intro":            "热门歌曲",
									"updateFrequency":  "每天",
									"pic":              "000test",
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

	charts, err := c.GetChartCategories(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(charts) != 1 {
		t.Fatalf("expected 1 chart, got %d", len(charts))
	}
	if charts[0].Name != "热歌榜" {
		t.Errorf("got %s", charts[0].Name)
	}
	if charts[0].ID != "qqmusic:chart:26" {
		t.Errorf("got %s", charts[0].ID)
	}
}

func TestGetHotComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{
					"commentlist": []any{
						map[string]any{
							"commentid":   "123",
							"nick":        "乐迷",
							"rootcomment": "好听!",
							"praisenum":   float64(100),
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

	comments, err := c.GetHotComments(t.Context(), "qqmusic:001abc", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].AuthorName != "乐迷" {
		t.Errorf("got %s", comments[0].AuthorName)
	}
	if comments[0].LikedCount != 100 {
		t.Errorf("got %d", comments[0].LikedCount)
	}
}

func TestHasPrefix(t *testing.T) {
	if !hasPrefix("http://test.com", "http") {
		t.Error("expected true")
	}
	if hasPrefix("abc", "xyz") {
		t.Error("expected false")
	}
}
