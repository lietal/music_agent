package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Conversation struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

var conversations = map[string]*Conversation{}

func createConversationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	conv := &Conversation{
		ID:     id,
		UserID: req.UserID,
	}
	conversations[id] = conv

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conv)
}

func listConversationsHandler(w http.ResponseWriter, r *http.Request) {
	convs := make([]*Conversation, 0, len(conversations))
	for _, c := range conversations {
		convs = append(convs, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convs)
}

func getConversationHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conv, ok := conversations[id]
	if !ok {
		http.Error(w, `{"error":"conversation not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}
