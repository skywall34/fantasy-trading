package handlers

import (
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type LeaderboardHandler struct {
	db *database.DB
}

func NewLeaderboardHandler(db *database.DB) *LeaderboardHandler {
	return &LeaderboardHandler{db: db}
}

func (h *LeaderboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "weekly"
	}

	// Get all public users
	publicUsers, err := h.db.GetAllPublicUsers()
	if err != nil {
		log.Printf("Error getting public users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Fetch live data from Alpaca for each user
	type userPerformance struct {
		UserID        int
		DisplayName   string
		Nickname      string
		AvatarURL     string
		CurrentEquity float64
		GainAmount    float64
		GainPercent   float64
		ShowAmounts   bool
	}

	var performances []userPerformance
	for _, u := range publicUsers {
		// Get user's session to fetch Alpaca data
		session, err := h.db.GetLatestSession(u.ID)
		if err != nil {
			log.Printf("No session found for user %d: %v", u.ID, err)
			continue
		}

		// Get decrypted API keys
		apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
		if err != nil {
			log.Printf("Failed to decrypt API keys for user %d: %v", u.ID, err)
			continue
		}

		// Fetch live data from Alpaca
		client := alpaca.NewClient(apiKey, apiSecret)
		account, err := client.GetAccount(r.Context())
		if err != nil {
			log.Printf("Failed to get Alpaca account for user %d: %v", u.ID, err)
			continue
		}

		// Parse account data
		equity, _ := strconv.ParseFloat(account.Equity, 64)

		// Calculate gain based on hardcoded starting equity (same as helpers.go)
		startingEquity := 100000.0
		totalGain := equity - startingEquity
		totalGainPct := 0.0
		if startingEquity > 0 {
			totalGainPct = (totalGain / startingEquity) * 100
		}

		displayName := "Unknown"
		if u.Nickname.Valid && u.Nickname.String != "" {
			displayName = u.Nickname.String
		} else if u.DisplayName.Valid && u.DisplayName.String != "" {
			displayName = u.DisplayName.String
		}

		nickname := ""
		if u.Nickname.Valid {
			nickname = u.Nickname.String
		}

		avatarURL := ""
		if u.AvatarURL.Valid {
			avatarURL = u.AvatarURL.String
		}

		performances = append(performances, userPerformance{
			UserID:        u.ID,
			DisplayName:   displayName,
			Nickname:      nickname,
			AvatarURL:     avatarURL,
			CurrentEquity: equity,
			GainAmount:    totalGain,
			GainPercent:   totalGainPct,
			ShowAmounts:   u.ShowAmounts,
		})
	}

	// Sort by gain percentage descending
	sort.Slice(performances, func(i, j int) bool {
		return performances[i].GainPercent > performances[j].GainPercent
	})

	// Convert to template entries with ranks
	templateEntries := make([]templates.LeaderboardEntryData, 0, len(performances))
	for rank, perf := range performances {
		templateEntries = append(templateEntries, templates.LeaderboardEntryData{
			UserID:        perf.UserID,
			DisplayName:   perf.DisplayName,
			Nickname:      perf.Nickname,
			AvatarURL:     perf.AvatarURL,
			CurrentEquity: perf.CurrentEquity,
			GainAmount:    perf.GainAmount,
			GainPercent:   perf.GainPercent,
			Rank:          rank + 1,
			ShowAmounts:   perf.ShowAmounts,
			IsCurrentUser: perf.UserID == userID,
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

	if err := templates.Leaderboard(templateUser, data).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering leaderboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
