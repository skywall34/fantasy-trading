package middleware

import (
	"context"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/database"
)

type contextKey string

const (
	UserIDKey     contextKey = "user_id"
	APIKeyKey     contextKey = "api_key"
	APISecretKey  contextKey = "api_secret"
	SessionIDKey  contextKey = "session_id"
)

// AuthMiddleware checks if the user is authenticated via session cookie
func AuthMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session cookie
			cookie, err := r.Cookie("session_id")
			if err != nil {
				// No session cookie - redirect to login
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			// Get session from database
			session, err := db.GetSessionByID(cookie.Value)
			if err != nil {
				// Invalid session - redirect to login
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			// Check if session is expired
			if session.IsExpired() {
				// Delete expired session
				_ = db.DeleteSession(session.ID)
				// Redirect to login
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
			ctx = context.WithValue(ctx, APIKeyKey, session.APIKey)
			ctx = context.WithValue(ctx, APISecretKey, session.APISecret)
			ctx = context.WithValue(ctx, SessionIDKey, session.ID)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID retrieves the user ID from the request context
func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}

// GetAPIKey retrieves the API key from the request context
func GetAPIKey(ctx context.Context) (string, bool) {
	apiKey, ok := ctx.Value(APIKeyKey).(string)
	return apiKey, ok
}

// GetAPISecret retrieves the API secret from the request context
func GetAPISecret(ctx context.Context) (string, bool) {
	apiSecret, ok := ctx.Value(APISecretKey).(string)
	return apiSecret, ok
}

// GetSessionID retrieves the session ID from the request context
func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(SessionIDKey).(string)
	return sessionID, ok
}
