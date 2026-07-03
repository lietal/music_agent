package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/db"
	"github.com/music-agent/music-agent/internal/event"
)

func testDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "postgres://music_agent:music_agent@127.0.0.1:5432/music_agent_test_api?sslmode=disable"
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("cannot connect: %v", err)
	}
	t.Cleanup(pool.Close)
	db.RunMigrations(ctx, url)
	return pool
}

func TestConversationsDB_CreateAndList(t *testing.T) {
	pool := testDBPool(t)
	h := &Handler{
		bus:       event.NewBus(),
		jwtSecret: []byte("test-secret"),
		db:        pool,
	}

	uid := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	pool.Exec(context.Background(),
		`INSERT INTO users (id, oauth_provider, oauth_id, display_name) VALUES ($1,'test','test','Test') ON CONFLICT DO NOTHING`,
		uid)

	user := &auth.UserInfo{UserID: uid, DisplayName: "Test"}
	ctx := auth.WithUser(context.Background(), user)

	body := `{"name":"Test Conv"}`
	req := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.createConversationHandler(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201 got %d: %s", rec.Code, rec.Body.String())
	}

	var createResp map[string]any
	json.NewDecoder(rec.Body).Decode(&createResp)
	convID := createResp["id"].(string)

	req2 := httptest.NewRequest("GET", "/api/conversations", nil).WithContext(ctx)
	rec2 := httptest.NewRecorder()
	h.listConversationsHandler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200 got %d", rec2.Code)
	}

	pool.Exec(context.Background(), "DELETE FROM conversations WHERE id=$1", convID)
}

func TestPlaylistDB_CreateAndList(t *testing.T) {
	pool := testDBPool(t)
	h := &Handler{
		bus:       event.NewBus(),
		jwtSecret: []byte("test-secret"),
		db:        pool,
	}

	uid := "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, oauth_provider, oauth_id, display_name) VALUES ($1,'test','test-xy2','Test') ON CONFLICT (id) DO NOTHING`,
		uid)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	user := &auth.UserInfo{UserID: uid}
	ctx := auth.WithUser(context.Background(), user)

	// Create playlist
	body := `{"name":"My Playlist","songs":[{"songId":"s1","title":"Song 1","artist":"Artist 1"}]}`
	req := httptest.NewRequest("POST", "/api/playlists", strings.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.createPlaylistHandler(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201 got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	pid := resp["id"].(string)

	// List
	req2 := httptest.NewRequest("GET", "/api/playlists", nil).WithContext(ctx)
	rec2 := httptest.NewRecorder()
	h.listPlaylistsHandler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200 got %d", rec2.Code)
	}

	// Get
	req3 := httptest.NewRequest("GET", "/api/playlists/"+pid, nil).WithContext(ctx)
	req3 = chiSetURLParam(req3, "id", pid)
	rec3 := httptest.NewRecorder()
	h.getPlaylistHandler(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("get: expected 200 got %d", rec3.Code)
	}

	pool.Exec(context.Background(), "DELETE FROM playlist_songs WHERE playlist_id=$1", pid)
	pool.Exec(context.Background(), "DELETE FROM playlists WHERE id=$1", pid)
}

func TestPlaylistDB_AddSong(t *testing.T) {
	pool := testDBPool(t)
	h := &Handler{db: pool}

	uid := "c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	pool.Exec(context.Background(),
		`INSERT INTO users (id, oauth_provider, oauth_id, display_name) VALUES ($1,'test','t3','T') ON CONFLICT (id) DO NOTHING`, uid)
	user := &auth.UserInfo{UserID: uid}
	ctx := auth.WithUser(context.Background(), user)

	body := `{"name":"P"}`
	req := httptest.NewRequest("POST", "/api/playlists", strings.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.createPlaylistHandler(rec, req)
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	pid := resp["id"].(string)

	addBody := `{"songId":"s2","title":"Song 2","artist":"Artist 2"}`
	req2 := httptest.NewRequest("POST", "/api/playlists/"+pid+"/songs", strings.NewReader(addBody)).WithContext(ctx)
	req2 = chiSetURLParam(req2, "id", pid)
	rec2 := httptest.NewRecorder()
	h.addToPlaylistHandler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("add song: expected 200 got %d: %s", rec2.Code, rec2.Body.String())
	}

	pool.Exec(context.Background(), "DELETE FROM playlist_songs WHERE playlist_id=$1", pid)
	pool.Exec(context.Background(), "DELETE FROM playlists WHERE id=$1", pid)
}

func TestPlaylistAdd_Unauthorized(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("POST", "/api/playlists/any/songs", strings.NewReader(`{"songId":"s"}`))
	req = chiSetURLParam(req, "id", "any")
	rec := httptest.NewRecorder()
	h.addToPlaylistHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestPlaylistAdd_BadBody(t *testing.T) {
	pool := testDBPool(t)
	h := &Handler{db: pool}
	uid := "d1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	pool.Exec(context.Background(),
		`INSERT INTO users (id, oauth_provider, oauth_id, display_name) VALUES ($1,'test','t4','T') ON CONFLICT (id) DO NOTHING`, uid)
	user := &auth.UserInfo{UserID: uid}
	ctx := auth.WithUser(context.Background(), user)
	req := httptest.NewRequest("POST", "/api/playlists/any/songs", strings.NewReader(`bad`)).WithContext(ctx)
	req = chiSetURLParam(req, "id", "any")
	rec := httptest.NewRecorder()
	h.addToPlaylistHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestPlaylistDelete_DB(t *testing.T) {
	pool := testDBPool(t)
	h := &Handler{db: pool}
	uid := "e1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	pool.Exec(context.Background(),
		`INSERT INTO users (id, oauth_provider, oauth_id, display_name) VALUES ($1,'test','t5','T') ON CONFLICT (id) DO NOTHING`, uid)
	user := &auth.UserInfo{UserID: uid}
	ctx := auth.WithUser(context.Background(), user)
	body := `{"name":"Temp"}`
	req := httptest.NewRequest("POST", "/api/playlists", strings.NewReader(body)).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.createPlaylistHandler(rec, req)
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	pid := resp["id"].(string)

	req2 := httptest.NewRequest("DELETE", "/api/playlists/"+pid, nil)
	req2 = chiSetURLParam(req2, "id", pid)
	rec2 := httptest.NewRecorder()
	h.deletePlaylistHandler(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec2.Code)
	}
}
