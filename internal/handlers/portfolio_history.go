package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

type PortfolioHistoryHandler struct{}

func NewPortfolioHistoryHandler() *PortfolioHistoryHandler {
	return &PortfolioHistoryHandler{}
}

type PortfolioHistoryResponse struct {
	Timestamps []int64   `json:"timestamps"`
	Equities   []float64 `json:"equities"`
}

func (h *PortfolioHistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	period := r.URL.Query().Get("period")
	timeframe := r.URL.Query().Get("timeframe")

	// Validate parameters
	if period == "" || timeframe == "" {
		http.Error(w, "Missing period or timeframe parameter", http.StatusBadRequest)
		return
	}

	// Create Alpaca client
	alpacaClient := alpaca.NewClient(apiKey, apiSecret)
	ctx := context.Background()

	// Get portfolio history
	history, err := alpacaClient.GetPortfolioHistory(ctx, period, timeframe)
	if err != nil {
		log.Printf("Failed to get portfolio history: %v", err)
		http.Error(w, "Failed to get portfolio history", http.StatusInternalServerError)
		return
	}

	// Build response
	response := PortfolioHistoryResponse{
		Timestamps: history.Timestamp,
		Equities:   history.Equity,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
