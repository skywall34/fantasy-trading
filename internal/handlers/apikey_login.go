package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/templates"
)

type APIKeyLoginHandler struct {
	db *database.DB
}

func NewAPIKeyLoginHandler(db *database.DB) *APIKeyLoginHandler {
	return &APIKeyLoginHandler{db: db}
}

func (h *APIKeyLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Show login page
		templates.LoginPage().Render(r.Context(), w)
		return
	}

	if r.Method == http.MethodPost {
		// Process login
		h.handleLogin(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *APIKeyLoginHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	apiKey := r.FormValue("api_key")
	apiSecret := r.FormValue("api_secret")

	if apiKey == "" || apiSecret == "" {
		http.Error(w, "API key and secret are required", http.StatusBadRequest)
		return
	}

	// Validate credentials by making a test API call
	alpacaClient := alpaca.NewClient(apiKey, apiSecret)
	ctx := context.Background()

	account, err := alpacaClient.GetAccount(ctx)
	if err != nil {
		log.Printf("Failed to validate API credentials: %v", err)
		errorMsg := "Invalid API credentials. Please check your API key and secret."
		templates.LoginPageWithError(errorMsg).Render(r.Context(), w)
		return
	}

	// Generate display name from account ID (last 8 characters)
	displayName := "User-" + account.ID[len(account.ID)-8:]
	if len(account.ID) < 8 {
		displayName = "User-" + account.ID
	}

	// Create or get user
	user, err := h.db.CreateUser(account.ID, nil, displayName)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour session
	_, err = h.db.CreateSession(sessionID, user.ID, apiKey, apiSecret, expiresAt)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Delete old sessions for this user (keep only the current one)
	if err := h.db.DeleteOldSessionsForUser(user.ID); err != nil {
		log.Printf("Failed to delete old sessions: %v", err)
		// Don't return error, this is not critical
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
