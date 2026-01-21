package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/cache"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type UserHandler struct {
	db    *database.DB
	cache *cache.Cache
}

func NewUserHandler(db *database.DB) *UserHandler {
	return &UserHandler{db: db, cache: nil}
}

func (h *UserHandler) SetCache(c *cache.Cache) {
	h.cache = c
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
		// Try to use cache for account data
		if h.cache != nil {
			cacheKey := fmt.Sprintf("account:%d", profileUserID)

			// Define refresh function
			refreshFunc := func(ctx context.Context) (any, error) {
				apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
				if err != nil {
					return nil, err
				}
				client := alpaca.NewClient(apiKey, apiSecret)
				return client.GetAccount(ctx)
			}

			// Get account data with 60s TTL (on-demand refresh, no background)
			if data, err := h.cache.GetOrSetWithRefresh(cacheKey, 60*time.Second, refreshFunc); err == nil {
				account := data.(*alpaca.Account)
				accountData := parseAccountData(account)
				performanceData = templates.PerformanceData{
					CurrentEquity: accountData.Equity,
					GainAmount:    accountData.TotalGain,
					GainPercent:   accountData.TotalGainPct,
				}
			}

			// Get positions (not cached - fetched on demand)
			apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
			if err == nil {
				client := alpaca.NewClient(apiKey, apiSecret)
				alpacaPositions, err := client.GetPositions(r.Context())
				if err == nil {
					positions = convertPositionsToTemplateData(alpacaPositions)
				}
			}
		} else {
			// Fallback to direct API call if cache not available
			apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
			if err == nil {
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
	}

	// Get recent activities from Alpaca
	recentActivities := make([]templates.ActivityData, 0)
	if session != nil {
		var alpacaActivities []alpaca.Activity

		// Try to use cache for activities
		if h.cache != nil {
			cacheKey := fmt.Sprintf("activities:%d", profileUserID)

			// Define refresh function
			refreshFunc := func(ctx context.Context) (any, error) {
				apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
				if err != nil {
					return nil, err
				}
				client := alpaca.NewClient(apiKey, apiSecret)
				return client.GetActivities(ctx)
			}

			// Get activities with 30s TTL (on-demand refresh)
			if data, err := h.cache.GetOrSetWithRefresh(cacheKey, 30*time.Second, refreshFunc); err == nil {
				alpacaActivities = data.([]alpaca.Activity)
			}
		} else {
			// Fallback to direct API call if cache not available
			apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
			if err == nil {
				client := alpaca.NewClient(apiKey, apiSecret)
				alpacaActivities, _ = client.GetActivities(r.Context())
			}
		}

		// Convert up to 10 most recent activities
		for i, act := range alpacaActivities {
			if i >= 10 {
				break
			}

			qty, _ := strconv.ParseFloat(act.Qty, 64)
			price, _ := strconv.ParseFloat(act.Price, 64)

			action := "traded"
			if act.Side == "buy" {
				action = "bought"
			} else if act.Side == "sell" {
				action = "sold"
			}

			timeAgo := "recently"
			if act.TransactionTime != "" {
				transTime, err := time.Parse(time.RFC3339, act.TransactionTime)
				if err == nil {
					timeAgo = formatTimeAgo(transTime)
				}
			}

			displayName := getDisplayName(profileUser)

			recentActivities = append(recentActivities, templates.ActivityData{
				UserName: displayName,
				Action:   action,
				Symbol:   act.Symbol,
				Qty:      qty,
				Price:    price,
				TimeAgo:  timeAgo,
			})
		}
	}

	// Get user rank - for now set to 0 as it requires fetching all users
	// Users can see their rank on the leaderboard page
	var rank int = 0
	var totalUsers int = 0

	// Get follower/following counts
	followerCount, _ := h.db.GetFollowerCount(profileUserID)
	followingCount, _ := h.db.GetFollowingCount(profileUserID)

	// Check if current user is following the profile user
	isFollowing := false
	if !isOwnProfile {
		isFollowing, _ = h.db.IsFollowing(currentUserID, profileUserID)
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
		FollowerCount:    followerCount,
		FollowingCount:   followingCount,
		IsFollowing:      isFollowing,
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
