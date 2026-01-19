package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

type FollowHandler struct {
	db *database.DB
}

func NewFollowHandler(db *database.DB) *FollowHandler {
	return &FollowHandler{db: db}
}

// ServeHTTP handles POST /follow/{userID} and DELETE /follow/{userID}
func (h *FollowHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	currentUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse target user ID from form
	targetUserIDStr := r.FormValue("user_id")
	if targetUserIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent self-following
	if currentUserID == targetUserID {
		http.Error(w, "Cannot follow yourself", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Follow user
		err := h.db.FollowUser(currentUserID, targetUserID)
		if err != nil {
			log.Printf("Error following user: %v", err)
			http.Error(w, "Failed to follow user", http.StatusInternalServerError)
			return
		}

		// Return success with HTMX trigger to refresh follow button
		w.Header().Set("HX-Trigger", "followChanged")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))

	case http.MethodDelete:
		// Unfollow user
		err := h.db.UnfollowUser(currentUserID, targetUserID)
		if err != nil {
			log.Printf("Error unfollowing user: %v", err)
			http.Error(w, "Failed to unfollow user", http.StatusInternalServerError)
			return
		}

		// Return success with HTMX trigger to refresh follow button
		w.Header().Set("HX-Trigger", "followChanged")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
