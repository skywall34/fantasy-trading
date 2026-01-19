package handlers

import (
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

type LogoutHandler struct {
	db *database.DB
}

func NewLogoutHandler(db *database.DB) *LogoutHandler {
	return &LogoutHandler{db: db}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
