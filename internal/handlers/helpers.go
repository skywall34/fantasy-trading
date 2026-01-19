package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/templates"
)

// formatTimeAgo converts a time to a human-readable relative time string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1d ago"
	}
	return fmt.Sprintf("%dd ago", days)
}

// convertActivityToTemplateData converts a database Activity to template ActivityData
func convertActivityToTemplateData(db *database.DB, act database.Activity) (templates.ActivityData, error) {
	// Get user for this activity
	user, err := db.GetUserByID(act.UserID)
	if err != nil {
		return templates.ActivityData{}, err
	}

	userName := "Unknown"
	if user.Nickname.Valid {
		userName = user.Nickname.String
	} else if user.DisplayName.Valid {
		userName = user.DisplayName.String
	}

	action := "traded"
	if act.Side.Valid {
		action = act.Side.String
		switch action {
		case "buy":
			action = "bought"
		case "sell":
			action = "sold"
		}
	}

	symbol := "N/A"
	if act.Symbol.Valid {
		symbol = act.Symbol.String
	}

	qty := 0.0
	if act.Qty.Valid {
		qty = act.Qty.Float64
	}

	price := 0.0
	if act.Price.Valid {
		price = act.Price.Float64
	}

	timeAgo := "recently"
	if act.TransactionTime.Valid {
		timeAgo = formatTimeAgo(act.TransactionTime.Time)
	}

	return templates.ActivityData{
		UserName: userName,
		Action:   action,
		Symbol:   symbol,
		Qty:      qty,
		Price:    price,
		TimeAgo:  timeAgo,
	}, nil
}


// AccountData holds parsed account information
type AccountData struct {
	Equity        float64
	LastEquity    float64
	Cash          float64
	BuyingPower   float64
	TodaysGain    float64
	TodaysGainPct float64
	TotalGain     float64
	TotalGainPct  float64
}

// parseAccountData converts raw account data from Alpaca to structured account data with calculations
func parseAccountData(account *alpaca.Account) AccountData {
	equity, _ := strconv.ParseFloat(account.Equity, 64)
	lastEquity, _ := strconv.ParseFloat(account.LastEquity, 64)
	cash, _ := strconv.ParseFloat(account.Cash, 64)
	buyingPower, _ := strconv.ParseFloat(account.BuyingPower, 64)

	// Calculate today's gain
	todaysGain := equity - lastEquity
	todaysGainPct := 0.0
	if lastEquity > 0 {
		todaysGainPct = (todaysGain / lastEquity) * 100
	}

	// For now, use mock data for total gain (we'll calculate this properly later)
	totalGain := equity - 100000 // Assuming starting value of $100k
	totalGainPct := 0.0
	if equity > 0 {
		totalGainPct = (totalGain / 100000) * 100
	}

	return AccountData{
		Equity:        equity,
		LastEquity:    lastEquity,
		Cash:          cash,
		BuyingPower:   buyingPower,
		TodaysGain:    todaysGain,
		TodaysGainPct: todaysGainPct,
		TotalGain:     totalGain,
		TotalGainPct:  totalGainPct,
	}
}

// convertPositionsToTemplateData converts Alpaca positions to template data
func convertPositionsToTemplateData(positions []alpaca.Position) []templates.PositionData {
	positionData := make([]templates.PositionData, 0, len(positions))
	for _, pos := range positions {
		qty, _ := strconv.ParseFloat(pos.Qty, 64)
		avgEntryPrice, _ := strconv.ParseFloat(pos.AvgEntryPrice, 64)
		marketValue, _ := strconv.ParseFloat(pos.MarketValue, 64)
		unrealizedPL, _ := strconv.ParseFloat(pos.UnrealizedPL, 64)
		unrealizedPct, _ := strconv.ParseFloat(pos.UnrealizedPLPC, 64)

		positionData = append(positionData, templates.PositionData{
			Symbol:        pos.Symbol,
			Name:          pos.Symbol, // TODO: Get actual company name
			AssetClass:    pos.AssetClass,
			Qty:           qty,
			Price:         avgEntryPrice,
			MarketValue:   marketValue,
			UnrealizedPL:  unrealizedPL,
			UnrealizedPct: unrealizedPct * 100,
		})
	}
	return positionData
}
