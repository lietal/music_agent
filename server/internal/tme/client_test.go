package tme

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSetBaseURL(t *testing.T) {
	c := NewClient()
	c.SetBaseURL("http://test.example.com")
}

func TestSetCredential(t *testing.T) {
	c := NewClient()
	c.SetCredential("12345", "W_Xabcd")
}

func TestCall_Success(t *testing.T) {
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
								"mid":      "001abc",
								"name":     "晴天",
								"interval": float64(269),
								"singer":   []any{map[string]any{"name": "周杰伦"}},
								"album":    map[string]any{"name": "叶惠美", "pmid": "000MkMni19ClKG_5"},
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

	songs, err := c.SearchSongs(t.Context(), "周杰伦", 5)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if len(songs) != 1 {
		t.Fatalf("expected 1 song, got %d", len(songs))
	}
	if songs[0].Title != "晴天" {
		t.Errorf("expected 晴天, got %s", songs[0].Title)
	}
	if songs[0].ID != "qqmusic:001abc" {
		t.Errorf("expected qqmusic:001abc, got %s", songs[0].ID)
	}
	if len(songs[0].Artists) == 0 || songs[0].Artists[0] != "周杰伦" {
		t.Errorf("expected [周杰伦], got %v", songs[0].Artists)
	}
	if songs[0].Album != "叶惠美" {
		t.Errorf("expected 叶惠美, got %s", songs[0].Album)
	}
}

func TestCall_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(10001),
			"req_0": map[string]any{
				"code": float64(10001),
				"data": map[string]any{},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	_, err := c.SearchSongs(t.Context(), "test", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCall_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	_, err := c.SearchSongs(t.Context(), "test", 1)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestParseSongs_Empty(t *testing.T) {
	c := NewClient()
	songs := c.parseSongs(map[string]any{})
	if len(songs) != 0 {
		t.Errorf("expected 0 songs, got %d", len(songs))
	}
}

func TestParseSongs_FlatList(t *testing.T) {
	c := NewClient()
	songs := c.parseSongs(map[string]any{
		"songlist": []any{
			map[string]any{
				"mid":      "abc",
				"name":     "Test Song",
				"interval": float64(200),
				"singer":   []any{map[string]any{"name": "Artist"}},
			},
		},
	})
	if len(songs) != 1 {
		t.Fatalf("expected 1, got %d", len(songs))
	}
}

func TestParseSongs_SingerName(t *testing.T) {
	c := NewClient()
	songs := c.parseSongs(map[string]any{
		"list": []any{
			map[string]any{
				"mid":         "abc",
				"name":        "Song",
				"singer_name": "Direct Singer",
				"interval":    float64(180),
			},
		},
	})
	if len(songs[0].Artists) != 1 || songs[0].Artists[0] != "Direct Singer" {
		t.Errorf("got %v", songs[0].Artists)
	}
}

func TestParseSongs_BodyItemSong(t *testing.T) {
	c := NewClient()
	songs := c.parseSongs(map[string]any{
		"body": map[string]any{
			"item_song": []any{
				map[string]any{
					"mid":      "bodyMid",
					"name":     "Body Song",
					"interval": float64(300),
					"singer":   []any{map[string]any{"name": "Singer"}},
					"album":    map[string]any{"name": "Album", "pmid": "000test"},
				},
			},
		},
	})
	if len(songs) != 1 {
		t.Fatalf("expected 1, got %d", len(songs))
	}
}

func TestExtractMID(t *testing.T) {
	tests := []struct{ in, want string }{
		{"qqmusic:003abc", "003abc"},
		{"qqmusic:track:003abc", "003abc"},
		{"qqmusic:song:003abc", "003abc"},
		{"003abc", "003abc"},
	}
	for _, tt := range tests {
		got := extractMID(tt.in)
		if got != tt.want {
			t.Errorf("extractMID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildArtworkURL(t *testing.T) {
	if u := buildArtworkURL("000MkMni19ClKG_5"); u != "https://y.gtimg.cn/music/photo_new/T002R300x300M000/000MkMni19ClKG_5.jpg" {
		t.Errorf("got %s", u)
	}
	if u := buildArtworkURL(""); u != "" {
		t.Errorf("expected empty, got %s", u)
	}
	if u := buildArtworkURL("https://example.com/img.jpg"); u != "https://example.com/img.jpg" {
		t.Errorf("got %s", u)
	}
}

func TestGetHelperFunctions(t *testing.T) {
	m := map[string]any{"key": "value", "num": float64(42)}
	if getString(m, "key") != "value" {
		t.Error("getString failed")
	}
	if getInt(m, "num") != 42 {
		t.Error("getInt failed")
	}
	if getFloat(m, "num") != 42 {
		t.Error("getFloat failed")
	}
	if getString(m, "missing") != "" {
		t.Error("getString for missing key should be empty")
	}
	if getInt(m, "missing") != 0 {
		t.Error("getInt for missing should be 0")
	}
}

func TestCallRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": float64(0),
			"req_0": map[string]any{
				"code": float64(0),
				"data": map[string]any{"key": "value"},
			},
		})
	}))
	defer server.Close()

	c := NewClient()
	c.SetBaseURL(server.URL)

	raw, err := c.CallRaw(t.Context(), map[string]MusicuSubRequest{
		"req_0": {Module: "test", Method: "test", Param: map[string]any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGetStringField(t *testing.T) {
	m := map[string]any{"a": "val_a", "b": "val_b"}
	if s := getStringField(m, "x", "a", "b"); s != "val_a" {
		t.Errorf("got %s", s)
	}
	if s := getStringField(m, "x", "y"); s != "" {
		t.Errorf("got %s", s)
	}
}

func TestTruncateBytes(t *testing.T) {
	b := []byte("hello world")
	if got := truncateBytes(b, 5); string(got) != "hello" {
		t.Errorf("got %s", string(got))
	}
	if got := truncateBytes(b, 100); string(got) != "hello world" {
		t.Errorf("got %s", string(got))
	}
}
