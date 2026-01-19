package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type ActivityHandler struct {
	db *database.DB
}

func NewActivityHandler(db *database.DB) *ActivityHandler {
	return &ActivityHandler{db: db}
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
	// TODO: Implement proper pagination with offset

	var activities []database.Activity
	switch filter {
	case "all":
		activities, err = h.db.GetRecentActivities(limit + 1)
	case "following":
		// TODO: Implement following filter when we add user following feature
		activities, err = h.db.GetRecentActivities(limit + 1)
	default:
		activities, err = h.db.GetRecentActivities(limit + 1)
	}

	if err != nil {
		log.Printf("Error getting activities: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasMore := len(activities) > limit
	if hasMore {
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
