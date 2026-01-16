package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/database"
)

type Scheduler struct {
	db       *database.DB
	interval time.Duration
	stopChan chan bool
}

// NewScheduler creates a new sync scheduler
func NewScheduler(db *database.DB, intervalMinutes int) *Scheduler {
	return &Scheduler{
		db:       db,
		interval: time.Duration(intervalMinutes) * time.Minute,
		stopChan: make(chan bool),
	}
}

// Start begins the background sync job
func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	go s.syncAllUsers()

	for {
		select {
		case <-ticker.C:
			go s.syncAllUsers()
		case <-s.stopChan:
			log.Println("Stopping sync scheduler")
			return
		}
	}
}

// Stop stops the background sync job
func (s *Scheduler) Stop() {
	s.stopChan <- true
}

// syncAllUsers syncs data for all users
func (s *Scheduler) syncAllUsers() {
	log.Println("Starting background sync for all users...")

	// Get all sessions to sync their data
	sessions, err := s.db.GetAllActiveSessions()
	if err != nil {
		log.Printf("Failed to get active sessions: %v", err)
		return
	}

	for _, session := range sessions {
		if session.IsExpired() {
			continue
		}

		if err := s.syncUserData(session.UserID, session.APIKey, session.APISecret); err != nil {
			log.Printf("Failed to sync user %d: %v", session.UserID, err)
		}
	}

	log.Printf("Background sync completed for %d users", len(sessions))
}

// syncUserData syncs data for a single user
func (s *Scheduler) syncUserData(userID int, apiKey, apiSecret string) error {
	ctx := context.Background()
	client := alpaca.NewClient(apiKey, apiSecret)

	// Get account data
	account, err := client.GetAccount(ctx)
	if err != nil {
		return err
	}

	// Parse and store portfolio snapshot
	equity := parseFloat(account.Equity)
	cash := parseFloat(account.Cash)
	buyingPower := parseFloat(account.BuyingPower)
	lastEquity := parseFloat(account.LastEquity)
	profitLoss := equity - lastEquity
	profitLossPct := 0.0
	if lastEquity > 0 {
		profitLossPct = (profitLoss / lastEquity) * 100
	}

	if err := s.db.CreatePortfolioSnapshot(userID, equity, cash, buyingPower, profitLoss, profitLossPct); err != nil {
		log.Printf("Failed to create portfolio snapshot for user %d: %v", userID, err)
	}

	// Get and store positions
	positions, err := client.GetPositions(ctx)
	if err != nil {
		log.Printf("Failed to get positions for user %d: %v", userID, err)
	} else {
		// Delete old positions
		_ = s.db.DeletePositionsForUser(userID)

		// Store new positions
		for _, pos := range positions {
			qty := parseFloat(pos.Qty)
			avgEntry := parseFloat(pos.AvgEntryPrice)
			currentPrice := parseFloat(pos.CurrentPrice)
			marketValue := parseFloat(pos.MarketValue)
			costBasis := parseFloat(pos.CostBasis)
			unrealizedPL := parseFloat(pos.UnrealizedPL)
			unrealizedPct := parseFloat(pos.UnrealizedPLPC)
			changeToday := parseFloat(pos.ChangeToday)

			err := s.db.UpsertPosition(
				userID,
				pos.Symbol,
				pos.AssetClass,
				pos.Symbol, // Using symbol as name for now
				qty,
				avgEntry,
				currentPrice,
				marketValue,
				costBasis,
				unrealizedPL,
				unrealizedPct*100, // Convert to percentage
				changeToday,
				"", // No option details for now
			)
			if err != nil {
				log.Printf("Failed to upsert position %s for user %d: %v", pos.Symbol, userID, err)
			}
		}
	}

	// Get and store activities
	activities, err := client.GetActivities(ctx)
	if err != nil {
		log.Printf("Failed to get activities for user %d: %v", userID, err)
	} else {
		log.Printf("Fetched %d activities for user %d", len(activities), userID)
		for _, act := range activities {
			qty := parseFloat(act.Qty)
			price := parseFloat(act.Price)
			transTime, _ := time.Parse(time.RFC3339, act.TransactionTime)

			err := s.db.UpsertActivity(
				act.ID,
				userID,
				act.ActivityType,
				"us_equity", // Default to equity
				act.Symbol,
				act.Side,
				qty,
				price,
				transTime,
			)
			if err != nil {
				log.Printf("Failed to upsert activity %s for user %d: %v", act.ID, userID, err)
			}
		}
	}

	// Update last sync time
	return s.db.UpdateLastSync(userID)
}

// parseFloat safely parses a string to float64
func parseFloat(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}
