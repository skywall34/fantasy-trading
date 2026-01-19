package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type ReactionsHandler struct {
	db *database.DB
}

func NewReactionsHandler(db *database.DB) *ReactionsHandler {
	return &ReactionsHandler{db: db}
}

func (h *ReactionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/activities/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	activityID := parts[0]

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.toggleReaction(w, r, activityID, userID)
}

func (h *ReactionsHandler) toggleReaction(w http.ResponseWriter, r *http.Request, activityID string, userID int) {
	var req struct {
		Emoji string `json:"emoji"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	validEmojis := map[string]bool{
		"🚀": true,
		"💎": true,
		"📈": true,
		"📉": true,
		"🔥": true,
		"👀": true,
		"🤔": true,
		"💰": true,
	}

	if !validEmojis[req.Emoji] {
		http.Error(w, "Invalid emoji", http.StatusBadRequest)
		return
	}

	_, err := h.db.AddReaction(activityID, userID, req.Emoji)
	if err != nil {
		log.Printf("Error toggling reaction: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	reactionCounts, err := h.db.GetReactionCounts(activityID)
	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	userReactions, err := h.db.GetUserReactionsForActivity(activityID, userID)
	if err != nil {
		log.Printf("Error getting user reactions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	activity := templates.ActivityFeedItem{
		ID:             activityID,
		ReactionCounts: reactionCounts,
		UserReactions:  userReactions,
	}

	if err := templates.ReactionButtons(activity).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering reactions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
