package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/cache"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type ActivityHandler struct {
	db    *database.DB
	cache *cache.Cache
}

func NewActivityHandler(db *database.DB) *ActivityHandler {
	return &ActivityHandler{db: db, cache: nil}
}

func (h *ActivityHandler) SetCache(c *cache.Cache) {
	h.cache = c
}

func (h *ActivityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	filter := r.URL.Query().Get("filter")
	if filter == "" {
		filter = "all"
	}

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	offset := (page - 1) * limit

	// Fetch live activities from Alpaca for all users
	var activities []database.Activity
	var usersToFetch []int

	switch filter {
	case "all":
		// Get all public users
		publicUsers, err := h.db.GetAllPublicUsers()
		if err != nil {
			log.Printf("Error getting public users: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		for _, u := range publicUsers {
			usersToFetch = append(usersToFetch, u.ID)
		}
	case "following":
		// Get users that current user is following
		usersToFetch, err = h.db.GetFollowing(userID)
		if err != nil {
			log.Printf("Error getting following: %v", err)
			usersToFetch = []int{}
		}
	default:
		// Get all public users
		publicUsers, err := h.db.GetAllPublicUsers()
		if err != nil {
			log.Printf("Error getting public users: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		for _, u := range publicUsers {
			usersToFetch = append(usersToFetch, u.ID)
		}
	}

	// Fetch activities from Alpaca for each user
	activities = h.fetchActivitiesForUsers(r.Context(), usersToFetch)

	// Apply pagination
	start := offset
	end := offset + limit + 1
	hasMore := false

	if start >= len(activities) {
		activities = []database.Activity{}
	} else {
		if end > len(activities) {
			end = len(activities)
		} else {
			hasMore = true
		}
		activities = activities[start:end]
	}

	if hasMore && len(activities) > limit {
		activities = activities[:limit]
	}

	activityIDs := make([]string, 0, len(activities))
	for _, act := range activities {
		activityIDs = append(activityIDs, act.ID)
	}

	commentCounts, _ := h.db.GetCommentCountsForActivities(activityIDs)
	reactionCountsMap, _ := h.db.GetReactionCountsForActivities(activityIDs)
	userReactionsMap, _ := h.db.GetUserReactionsForActivities(activityIDs, userID)

	templateActivities := make([]templates.ActivityFeedItem, 0, len(activities))
	for _, act := range activities {
		if !act.Symbol.Valid || !act.Qty.Valid {
			continue
		}

		actUser, err := h.db.GetUserByID(act.UserID)
		if err != nil {
			continue
		}

		userName := "Unknown"
		nickname := ""
		if actUser.DisplayName.Valid && actUser.DisplayName.String != "" {
			userName = actUser.DisplayName.String
		}
		if actUser.Nickname.Valid && actUser.Nickname.String != "" {
			nickname = actUser.Nickname.String
		}

		action := "traded"
		if act.Side.Valid {
			action = act.Side.String
		}

		timeAgo := "recently"
		if act.TransactionTime.Valid {
			timeAgo = formatTimeAgo(act.TransactionTime.Time)
		}

		assetClass := ""
		if act.AssetClass.Valid {
			assetClass = act.AssetClass.String
		}

		price := 0.0
		if act.Price.Valid {
			price = act.Price.Float64
		}

		commentCount := commentCounts[act.ID]

		reactionCounts := reactionCountsMap[act.ID]
		if reactionCounts == nil {
			reactionCounts = make(map[string]int)
		}
		userReactions := userReactionsMap[act.ID]
		if userReactions == nil {
			userReactions = []string{}
		}

		templateActivities = append(templateActivities, templates.ActivityFeedItem{
			ID:             act.ID,
			UserID:         act.UserID,
			UserName:       userName,
			UserNickname:   nickname,
			UserAvatarURL:  "",
			Action:         action,
			Symbol:         act.Symbol.String,
			AssetClass:     assetClass,
			Qty:            act.Qty.Float64,
			Price:          price,
			TimeAgo:        timeAgo,
			CommentCount:   commentCount,
			ReactionCounts: reactionCounts,
			UserReactions:  userReactions,
		})
	}

	data := templates.ActivityFeedData{
		Activities: templateActivities,
		Filter:     filter,
		HasMore:    hasMore,
		Page:       page,
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := templates.ActivityContent(data).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering activity content: %v", err)
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

	if err := templates.Activity(templateUser, data).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering activity: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// fetchActivitiesForUsers fetches live activities from Alpaca for multiple users
func (h *ActivityHandler) fetchActivitiesForUsers(ctx context.Context, userIDs []int) []database.Activity {
	var allActivities []database.Activity

	for _, uid := range userIDs {
		var alpacaActivities []alpaca.Activity

		// Try to use cache if available
		if h.cache != nil {
			cacheKey := fmt.Sprintf("activities:%d", uid)

			// Define refresh function for this user
			userID := uid // Capture for closure
			refreshFunc := func(ctx context.Context) (any, error) {
				session, err := h.db.GetLatestSession(userID)
				if err != nil {
					return nil, err
				}

				apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
				if err != nil {
					return nil, err
				}

				client := alpaca.NewClient(apiKey, apiSecret)
				return client.GetActivities(ctx)
			}

			// Get or set with auto-refresh (30s TTL)
			data, err := h.cache.GetOrSetWithRefresh(cacheKey, 30*time.Second, refreshFunc)
			if err != nil {
				log.Printf("Failed to get activities for user %d: %v", uid, err)
				continue
			}

			alpacaActivities = data.([]alpaca.Activity)
		} else {
			// Fallback to direct API call if cache is not available
			session, err := h.db.GetLatestSession(uid)
			if err != nil {
				log.Printf("No session found for user %d: %v", uid, err)
				continue
			}

			apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
			if err != nil {
				log.Printf("Failed to decrypt API keys for user %d: %v", uid, err)
				continue
			}

			client := alpaca.NewClient(apiKey, apiSecret)
			alpacaActivities, err = client.GetActivities(ctx)
			if err != nil {
				log.Printf("Failed to get activities from Alpaca for user %d: %v", uid, err)
				continue
			}
		}

		// Convert to database.Activity format
		for _, act := range alpacaActivities {
			qty, _ := strconv.ParseFloat(act.Qty, 64)
			price, _ := strconv.ParseFloat(act.Price, 64)
			transTime, _ := time.Parse(time.RFC3339, act.TransactionTime)

			allActivities = append(allActivities, database.Activity{
				ID:              act.ID,
				UserID:          uid,
				ActivityType:    act.ActivityType,
				AssetClass:      database.NewNullString("us_equity"),
				Symbol:          database.NewNullString(act.Symbol),
				Side:            database.NewNullString(act.Side),
				Qty:             database.NewNullFloat64(qty),
				Price:           database.NewNullFloat64(price),
				TransactionTime: database.NewNullTime(transTime),
			})
		}
	}

	// Sort all activities by transaction time descending
	sort.Slice(allActivities, func(i, j int) bool {
		if !allActivities[i].TransactionTime.Valid || !allActivities[j].TransactionTime.Valid {
			return false
		}
		return allActivities[i].TransactionTime.Time.After(allActivities[j].TransactionTime.Time)
	})

	return allActivities
}
