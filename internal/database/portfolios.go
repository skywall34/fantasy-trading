package database

import (
	"database/sql"
	"time"
)

type PortfolioSnapshot struct {
	ID             int
	UserID         int
	Equity         float64
	Cash           sql.NullFloat64
	BuyingPower    sql.NullFloat64
	ProfitLoss     sql.NullFloat64
	ProfitLossPct  sql.NullFloat64
	SnapshotAt     time.Time
}

func (db *DB) CreatePortfolioSnapshot(userID int, equity, cash, buyingPower, profitLoss, profitLossPct float64) error {
	query := `
		INSERT INTO portfolio_snapshots (user_id, equity, cash, buying_power, profit_loss, profit_loss_pct)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, userID, equity, cash, buyingPower, profitLoss, profitLossPct)
	return err
}

func (db *DB) GetLatestSnapshot(userID int) (*PortfolioSnapshot, error) {
	query := `
		SELECT id, user_id, equity, cash, buying_power, profit_loss, profit_loss_pct, snapshot_at
		FROM portfolio_snapshots
		WHERE user_id = ?
		ORDER BY snapshot_at DESC
		LIMIT 1
	`

	var snapshot PortfolioSnapshot
	err := db.QueryRow(query, userID).Scan(
		&snapshot.ID,
		&snapshot.UserID,
		&snapshot.Equity,
		&snapshot.Cash,
		&snapshot.BuyingPower,
		&snapshot.ProfitLoss,
		&snapshot.ProfitLossPct,
		&snapshot.SnapshotAt,
	)
	if err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (db *DB) GetSnapshotHistory(userID int, startTime, endTime time.Time) ([]PortfolioSnapshot, error) {
	query := `
		SELECT id, user_id, equity, cash, buying_power, profit_loss, profit_loss_pct, snapshot_at
		FROM portfolio_snapshots
		WHERE user_id = ? AND snapshot_at BETWEEN ? AND ?
		ORDER BY snapshot_at ASC
	`

	rows, err := db.Query(query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []PortfolioSnapshot
	for rows.Next() {
		var snapshot PortfolioSnapshot
		err := rows.Scan(
			&snapshot.ID,
			&snapshot.UserID,
			&snapshot.Equity,
			&snapshot.Cash,
			&snapshot.BuyingPower,
			&snapshot.ProfitLoss,
			&snapshot.ProfitLossPct,
			&snapshot.SnapshotAt,
		)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// LeaderboardEntry represents a user's ranking on the leaderboard
type LeaderboardEntry struct {
	UserID       int
	DisplayName  string
	Nickname     string
	AvatarURL    string
	CurrentEquity float64
	StartEquity  float64
	GainAmount   float64
	GainPercent  float64
	Rank         int
	ShowAmounts  bool
}

func (db *DB) GetLeaderboardDaily() ([]LeaderboardEntry, error) {
	// Get rankings from yesterday's close (approx 24 hours ago) to now
	startTime := time.Now().Add(-24 * time.Hour)
	return db.getLeaderboard(startTime)
}

func (db *DB) GetLeaderboardWeekly() ([]LeaderboardEntry, error) {
	// Get rankings from Monday of this week
	now := time.Now()
	weekday := now.Weekday()
	daysToMonday := int(weekday - time.Monday)
	if daysToMonday < 0 {
		daysToMonday += 7
	}
	startTime := now.AddDate(0, 0, -daysToMonday).Truncate(24 * time.Hour)
	return db.getLeaderboard(startTime)
}

func (db *DB) GetLeaderboardMonthly() ([]LeaderboardEntry, error) {
	// Get rankings from first day of current month
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return db.getLeaderboard(startTime)
}

func (db *DB) GetLeaderboardAllTime() ([]LeaderboardEntry, error) {
	// Get rankings from account creation
	startTime := time.Time{} // Beginning of time
	return db.getLeaderboard(startTime)
}

// getLeaderboard is a helper function that calculates rankings for a given time period
func (db *DB) getLeaderboard(startTime time.Time) ([]LeaderboardEntry, error) {
	query := `
		WITH user_equity AS (
			SELECT
				u.id as user_id,
				u.display_name,
				u.nickname,
				u.avatar_url,
				u.show_amounts,
				COALESCE((SELECT equity FROM portfolio_snapshots WHERE user_id = u.id ORDER BY snapshot_at DESC LIMIT 1), 0) as current_equity,
				COALESCE((
					SELECT equity
					FROM portfolio_snapshots
					WHERE user_id = u.id
					AND snapshot_at >= ?
					ORDER BY snapshot_at ASC
					LIMIT 1
				), COALESCE((SELECT equity FROM portfolio_snapshots WHERE user_id = u.id ORDER BY snapshot_at ASC LIMIT 1), 0)) as start_equity
			FROM users u
			WHERE u.is_public = 1
		)
		SELECT
			user_id,
			display_name,
			nickname,
			avatar_url,
			current_equity,
			start_equity,
			(current_equity - start_equity) as gain_amount,
			CASE
				WHEN start_equity > 0 THEN ((current_equity - start_equity) / start_equity * 100.0)
				ELSE 0
			END as gain_percent,
			show_amounts
		FROM user_equity
		WHERE start_equity > 0
		ORDER BY gain_percent DESC
	`

	rows, err := db.Query(query, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		var displayName, nickname, avatarURL sql.NullString

		err := rows.Scan(
			&entry.UserID,
			&displayName,
			&nickname,
			&avatarURL,
			&entry.CurrentEquity,
			&entry.StartEquity,
			&entry.GainAmount,
			&entry.GainPercent,
			&entry.ShowAmounts,
		)
		if err != nil {
			return nil, err
		}

		entry.DisplayName = displayName.String
		entry.Nickname = nickname.String
		entry.AvatarURL = avatarURL.String
		entry.Rank = rank
		rank++

		entries = append(entries, entry)
	}

	return entries, nil
}
