package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/cache"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

type LogoutHandler struct {
	db    *database.DB
	cache *cache.Cache
}

func NewLogoutHandler(db *database.DB) *LogoutHandler {
	return &LogoutHandler{db: db, cache: nil}
}

func (h *LogoutHandler) SetCache(c *cache.Cache) {
	h.cache = c
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (if authenticated)
	userID, hasUserID := middleware.GetUserID(r.Context())

	// Get session ID from context (set by auth middleware if available)
	sessionID, ok := middleware.GetSessionID(r.Context())
	if !ok {
		// Try to get it from cookie directly if not in context
		cookie, err := r.Cookie("session_id")
		if err == nil {
			sessionID = cookie.Value
		}
	}

	// Delete session from database if we have a session ID
	if sessionID != "" {
		if err := h.db.DeleteSession(sessionID); err != nil {
			log.Printf("Failed to delete session: %v", err)
			// Continue anyway - we'll still clear the cookie
		}
	}

	// Invalidate cache entries for this user
	if hasUserID && h.cache != nil {
		accountKey := fmt.Sprintf("account:%d", userID)
		activitiesKey := fmt.Sprintf("activities:%d", userID)
		h.cache.Delete(accountKey)
		h.cache.Delete(activitiesKey)
		log.Printf("Invalidated cache for user %d on logout", userID)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Immediate expiration
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}
