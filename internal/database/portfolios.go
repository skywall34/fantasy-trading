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

// CreatePortfolioSnapshot creates a new portfolio snapshot
func (db *DB) CreatePortfolioSnapshot(userID int, equity, cash, buyingPower, profitLoss, profitLossPct float64) error {
	query := `
		INSERT INTO portfolio_snapshots (user_id, equity, cash, buying_power, profit_loss, profit_loss_pct)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query, userID, equity, cash, buyingPower, profitLoss, profitLossPct)
	return err
}

// GetLatestSnapshot gets the most recent portfolio snapshot for a user
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

// GetSnapshotHistory gets portfolio snapshots for a user within a time range
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
