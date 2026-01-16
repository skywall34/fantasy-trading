package database

import (
	"database/sql"
	"time"
)

type Position struct {
	ID               int
	UserID           int
	Symbol           string
	AssetClass       string
	AssetName        sql.NullString
	Qty              float64
	AvgEntryPrice    sql.NullFloat64
	CurrentPrice     sql.NullFloat64
	MarketValue      sql.NullFloat64
	CostBasis        sql.NullFloat64
	UnrealizedPL     sql.NullFloat64
	UnrealizedPLPct  sql.NullFloat64
	ChangeToday      sql.NullFloat64
	OptionDetails    sql.NullString
	UpdatedAt        time.Time
}

// UpsertPosition creates or updates a position
func (db *DB) UpsertPosition(userID int, symbol, assetClass, assetName string, qty, avgEntryPrice, currentPrice, marketValue, costBasis, unrealizedPL, unrealizedPLPct, changeToday float64, optionDetails string) error {
	query := `
		INSERT INTO positions (
			user_id, symbol, asset_class, asset_name, qty, avg_entry_price, current_price,
			market_value, cost_basis, unrealized_pl, unrealized_pl_pct, change_today, option_details
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, symbol) DO UPDATE SET
			asset_class = excluded.asset_class,
			asset_name = excluded.asset_name,
			qty = excluded.qty,
			avg_entry_price = excluded.avg_entry_price,
			current_price = excluded.current_price,
			market_value = excluded.market_value,
			cost_basis = excluded.cost_basis,
			unrealized_pl = excluded.unrealized_pl,
			unrealized_pl_pct = excluded.unrealized_pl_pct,
			change_today = excluded.change_today,
			option_details = excluded.option_details,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.Exec(query, userID, symbol, assetClass, assetName, qty, avgEntryPrice, currentPrice, marketValue, costBasis, unrealizedPL, unrealizedPLPct, changeToday, optionDetails)
	return err
}

// GetPositionsByUser retrieves all positions for a user
func (db *DB) GetPositionsByUser(userID int) ([]Position, error) {
	query := `
		SELECT id, user_id, symbol, asset_class, asset_name, qty, avg_entry_price, current_price,
		       market_value, cost_basis, unrealized_pl, unrealized_pl_pct, change_today, option_details, updated_at
		FROM positions
		WHERE user_id = ?
		ORDER BY market_value DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var pos Position
		err := rows.Scan(
			&pos.ID,
			&pos.UserID,
			&pos.Symbol,
			&pos.AssetClass,
			&pos.AssetName,
			&pos.Qty,
			&pos.AvgEntryPrice,
			&pos.CurrentPrice,
			&pos.MarketValue,
			&pos.CostBasis,
			&pos.UnrealizedPL,
			&pos.UnrealizedPLPct,
			&pos.ChangeToday,
			&pos.OptionDetails,
			&pos.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

// GetPositionsByAssetClass retrieves positions filtered by asset class
func (db *DB) GetPositionsByAssetClass(userID int, assetClass string) ([]Position, error) {
	query := `
		SELECT id, user_id, symbol, asset_class, asset_name, qty, avg_entry_price, current_price,
		       market_value, cost_basis, unrealized_pl, unrealized_pl_pct, change_today, option_details, updated_at
		FROM positions
		WHERE user_id = ? AND asset_class = ?
		ORDER BY market_value DESC
	`

	rows, err := db.Query(query, userID, assetClass)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var pos Position
		err := rows.Scan(
			&pos.ID,
			&pos.UserID,
			&pos.Symbol,
			&pos.AssetClass,
			&pos.AssetName,
			&pos.Qty,
			&pos.AvgEntryPrice,
			&pos.CurrentPrice,
			&pos.MarketValue,
			&pos.CostBasis,
			&pos.UnrealizedPL,
			&pos.UnrealizedPLPct,
			&pos.ChangeToday,
			&pos.OptionDetails,
			&pos.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

// DeletePositionsForUser deletes all positions for a user (for resyncing)
func (db *DB) DeletePositionsForUser(userID int) error {
	query := `DELETE FROM positions WHERE user_id = ?`
	_, err := db.Exec(query, userID)
	return err
}
