package handlers

import (
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

// SettingsHandler handles user settings page
type SettingsHandler struct {
	db *database.DB
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(db *database.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

// ServeHTTP handles settings page requests
func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok || userID == 0 {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get display name and initials for template
	displayName := "User"
	if user.DisplayName.Valid {
		displayName = user.DisplayName.String
	}

	// Get initials
	initials := ""
	if len(displayName) > 0 {
		initials = string(displayName[0])
	}

	// Get current nickname
	currentNickname := ""
	if user.Nickname.Valid {
		currentNickname = user.Nickname.String
	}

	// Create template user
	templateUser := &templates.User{
		ID:          user.ID,
		DisplayName: displayName,
		Initials:    initials,
	}

	// Render settings template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = templates.Settings(templateUser, currentNickname).Render(r.Context(), w)
	if err != nil {
		log.Printf("Failed to render settings template: %v", err)
		http.Error(w, "Failed to render settings", http.StatusInternalServerError)
	}
}
