package handlers

import (
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/templates"
)

type LeaderboardHandler struct {
	db *database.DB
}

func NewLeaderboardHandler(db *database.DB) *LeaderboardHandler {
	return &LeaderboardHandler{db: db}
}

func (h *LeaderboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
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

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "weekly"
	}

	var entries []database.LeaderboardEntry
	switch period {
	case "daily":
		entries, err = h.db.GetLeaderboardDaily()
	case "weekly":
		entries, err = h.db.GetLeaderboardWeekly()
	case "monthly":
		entries, err = h.db.GetLeaderboardMonthly()
	case "all":
		entries, err = h.db.GetLeaderboardAllTime()
	default:
		entries, err = h.db.GetLeaderboardWeekly()
		period = "weekly"
	}

	if err != nil {
		log.Printf("Error getting leaderboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	templateEntries := make([]templates.LeaderboardEntryData, 0, len(entries))
	for _, entry := range entries {
		templateEntries = append(templateEntries, templates.LeaderboardEntryData{
			UserID:        entry.UserID,
			DisplayName:   entry.DisplayName,
			Nickname:      entry.Nickname,
			AvatarURL:     entry.AvatarURL,
			CurrentEquity: entry.CurrentEquity,
			GainAmount:    entry.GainAmount,
			GainPercent:   entry.GainPercent,
			Rank:          entry.Rank,
			ShowAmounts:   entry.ShowAmounts,
			IsCurrentUser: entry.UserID == userID,
		})
	}

	data := templates.LeaderboardData{
		Entries:       templateEntries,
		CurrentUserID: userID,
		Period:        period,
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := templates.LeaderboardContent(data).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering leaderboard content: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	displayName := user.DisplayName.String
	if displayName == "" && user.Nickname.String != "" {
		displayName = user.Nickname.String
	}
	if displayName == "" && user.Email.String != "" {
		displayName = user.Email.String
	}

	initials := "U"
	if len(displayName) > 0 {
		initials = string(displayName[0])
	}

	templateUser := &templates.User{
		ID:          user.ID,
		DisplayName: displayName,
		Initials:    initials,
	}

	if err := templates.Leaderboard(templateUser, data).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering leaderboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
