package handlers

import (
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type SearchHandler struct {
	db *database.DB
}

func NewSearchHandler(db *database.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.db.GetUserByID(userID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	searchQuery := r.URL.Query().Get("q")

	var results []templates.SearchResult
	if searchQuery != "" && len(searchQuery) >= 2 {
		users, err := h.db.SearchUsers(searchQuery, 20)
		if err != nil {
			log.Printf("Error searching users: %v", err)
		} else {
			for _, u := range users {
				displayName := "Unknown"
				if u.Nickname.Valid && u.Nickname.String != "" {
					displayName = u.Nickname.String
				} else if u.DisplayName.Valid && u.DisplayName.String != "" {
					displayName = u.DisplayName.String
				}

				avatarURL := ""
				if u.AvatarURL.Valid {
					avatarURL = u.AvatarURL.String
				}

				// Get follower/following status
				isFollowing := false
				if u.ID != userID {
					isFollowing, _ = h.db.IsFollowing(userID, u.ID)
				}

				followerCount, _ := h.db.GetFollowerCount(u.ID)

				results = append(results, templates.SearchResult{
					UserID:        u.ID,
					DisplayName:   displayName,
					AvatarURL:     avatarURL,
					IsFollowing:   isFollowing,
					FollowerCount: followerCount,
					IsSelf:        u.ID == userID,
				})
			}
		}
	}

	data := templates.SearchData{
		Query:   searchQuery,
		Results: results,
	}

	// If HTMX request, return only results
	if r.Header.Get("HX-Request") == "true" {
		if err := templates.SearchResults(data).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering search results: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	displayName := getDisplayName(user)

	initials := "U"
	if len(displayName) > 0 {
		initials = string(displayName[0])
	}

	templateUser := &templates.User{
		ID:          user.ID,
		DisplayName: displayName,
		Initials:    initials,
	}

	if err := templates.SearchPage(templateUser, data).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering search page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
