package api

import (
	"net/http"

	"github.com/music-agent/music-agent/internal/auth"
)

func JWTAuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return auth.JWTAuth(secret)
}
