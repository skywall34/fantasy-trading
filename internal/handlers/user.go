package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type UserHandler struct {
	db *database.DB
}

func NewUserHandler(db *database.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	currentUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract user ID from path
	path := strings.TrimPrefix(r.URL.Path, "/user/")
	profileUserID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get current user info for layout
	currentUser, err := h.db.GetUserByID(currentUserID)
	if err != nil {
		log.Printf("Error getting current user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get profile user
	profileUser, err := h.db.GetUserByID(profileUserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if profile is public or if viewing own profile
	isOwnProfile := currentUserID == profileUserID
	if !profileUser.IsPublic && !isOwnProfile {
		http.Error(w, "Profile is private", http.StatusForbidden)
		return
	}

	// Get user's session to fetch Alpaca data
	session, err := h.db.GetLatestSession(profileUserID)
	if err != nil {
		log.Printf("No session found for user %d: %v", profileUserID, err)
		// Continue without live data
	}

	var positions []templates.PositionData
	var performanceData templates.PerformanceData

	if session != nil {
		// Get decrypted API keys
		apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
		if err == nil {
			// Fetch live data from Alpaca
			client := alpaca.NewClient(apiKey, apiSecret)

			// Get account info
			account, err := client.GetAccount(r.Context())
			if err == nil {
				accountData := parseAccountData(account)
				performanceData = templates.PerformanceData{
					CurrentEquity: accountData.Equity,
					GainAmount:    accountData.TotalGain,
					GainPercent:   accountData.TotalGainPct,
				}

				// Get positions
				alpacaPositions, err := client.GetPositions(r.Context())
				if err == nil {
					positions = convertPositionsToTemplateData(alpacaPositions)
				}
			}
		}
	}

	// Get recent activities
	activities, err := h.db.GetUserRecentActivities(profileUserID, 10)
	if err != nil {
		log.Printf("Error getting user activities: %v", err)
	}

	recentActivities := make([]templates.ActivityData, 0)
	for _, act := range activities {
		if !act.Symbol.Valid || !act.Qty.Valid {
			continue
		}
		activityData, err := convertActivityToTemplateData(h.db, act)
		if err != nil {
			continue
		}
		recentActivities = append(recentActivities, activityData)
	}

	// Get user rank
	var rank int
	var totalUsers int
	leaderboard, err := h.db.GetLeaderboardAllTime()
	if err == nil {
		totalUsers = len(leaderboard)
		for i, entry := range leaderboard {
			if entry.UserID == profileUserID {
				rank = i + 1
				break
			}
		}
	}

	// Build profile data
	displayName := getDisplayName(profileUser)
	nickname := profileUser.Nickname.String

	avatarURL := ""
	if profileUser.AvatarURL.Valid {
		avatarURL = profileUser.AvatarURL.String
	}

	memberSince := profileUser.CreatedAt.Format("January 2006")

	userProfile := templates.UserProfile{
		ID:          profileUser.ID,
		DisplayName: displayName,
		Nickname:    nickname,
		AvatarURL:   avatarURL,
		IsPublic:    profileUser.IsPublic,
		ShowAmounts: profileUser.ShowAmounts,
		MemberSince: memberSince,
	}

	data := templates.UserProfileData{
		ProfileUser:      userProfile,
		IsOwnProfile:     isOwnProfile,
		CurrentUserID:    currentUserID,
		Rank:             rank,
		TotalUsers:       totalUsers,
		Positions:        positions,
		RecentActivities: recentActivities,
		PerformanceData:  performanceData,
	}

	// Prepare template user
	currentDisplayName := getDisplayName(currentUser)

	initials := "U"
	if len(currentDisplayName) > 0 {
		initials = string(currentDisplayName[0])
	}

	templateUser := &templates.User{
		ID:          currentUser.ID,
		DisplayName: currentDisplayName,
		Initials:    initials,
	}

	// Render template
	if err := templates.UserProfilePage(templateUser, data).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering user profile: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
