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

type DashboardContentHandler struct {
	db *database.DB
}

func NewDashboardContentHandler(db *database.DB) *DashboardContentHandler {
	return &DashboardContentHandler{db: db}
}

func (h *DashboardContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get API credentials from context
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

	// Get recent activity from database for this user
	recentActivity := getRecentActivityData(h.db, userID, 10)

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

	// Render only the content partial
	templates.DashboardContent(data).Render(r.Context(), w)
}
