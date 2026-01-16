package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

// UpdateProfileHandler handles user profile updates
type UpdateProfileHandler struct {
	db *database.DB
}

// NewUpdateProfileHandler creates a new update profile handler
func NewUpdateProfileHandler(db *database.DB) *UpdateProfileHandler {
	return &UpdateProfileHandler{db: db}
}

// ServeHTTP handles profile update requests
func (h *UpdateProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok || userID == 0 {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get nickname from form
	nickname := strings.TrimSpace(r.FormValue("nickname"))

	// Validate nickname
	if len(nickname) > 50 {
		http.Error(w, "Nickname must be 50 characters or less", http.StatusBadRequest)
		return
	}

	// Update nickname in database
	var nullNickname sql.NullString
	if nickname != "" {
		nullNickname = sql.NullString{String: nickname, Valid: true}
	}

	err = h.db.UpdateUserNickname(userID, nullNickname)
	if err != nil {
		log.Printf("Failed to update nickname: %v", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true, "message": "Profile updated successfully"}`))
}
