package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/templates"
)

type DashboardHandler struct {
	db *database.DB
}

func NewDashboardHandler(db *database.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get user ID and access token from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiKey, ok := middleware.GetAPIKey(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiSecret, ok := middleware.GetAPISecret(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create Alpaca client
	alpacaClient := alpaca.NewClient(apiKey, apiSecret)
	ctx := context.Background()

	// Get account info
	account, err := alpacaClient.GetAccount(ctx)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		http.Error(w, "Failed to get account data", http.StatusInternalServerError)
		return
	}

	// Get positions
	positions, err := alpacaClient.GetPositions(ctx)
	if err != nil {
		log.Printf("Failed to get positions: %v", err)
		// Continue even if positions fail
		positions = []alpaca.Position{}
	}

	// Get portfolio history
	portfolioHistory, err := alpacaClient.GetPortfolioHistory(ctx, "1m", "1D")
	if err != nil {
		log.Printf("Failed to get portfolio history: %v", err)
		portfolioHistory = &alpaca.PortfolioHistory{}
	}

	// Parse account data
	accountData := parseAccountData(account)

	// Convert positions to template data
	positionData := convertPositionsToTemplateData(positions)

	// Get recent activities from database for this user
	recentActivity := getRecentActivityData(h.db, userID, 10)

	// Create template user
	templateUser := &templates.User{
		ID:          user.ID,
		DisplayName: getDisplayName(user),
		Initials:    getInitials(user),
	}

	// Create dashboard data
	data := templates.DashboardData{
		PortfolioValue: accountData.Equity,
		TodaysGain:     accountData.TodaysGain,
		TodaysGainPct:  accountData.TodaysGainPct,
		TotalGain:      accountData.TotalGain,
		TotalGainPct:   accountData.TotalGainPct,
		BuyingPower:    accountData.BuyingPower,
		Cash:           accountData.Cash,
		Positions:      positionData,
		RecentActivity: recentActivity,
		PortfolioHistory: templates.PortfolioHistoryData{
			Timestamps: portfolioHistory.Timestamp,
			Equities:   portfolioHistory.Equity,
		},
	}

	// Render template
	templates.Dashboard(templateUser, data).Render(r.Context(), w)
}
